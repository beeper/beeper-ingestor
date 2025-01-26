// Based on gomuks - https://github.com/tulir/gomuks
// Copyright (C) 2024 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	_ "go.mau.fi/util/dbutil/litestream"
	flag "maunium.net/go/mauflag"
	"maunium.net/go/mautrix"

	"go.mau.fi/util/exerrors"

	"crypto/sha256"
	"encoding/base64"

	"github.com/beeper/beeper-mc-ingestor/web"
	"go.mau.fi/gomuks/pkg/gomuks"
	"go.mau.fi/gomuks/pkg/hicli"
)

var (
	Tag       = "unknown"
	Commit    = "unknown"
	BuildTime = "unknown"
)

const StaticVersion = "0.4.0"
const URL = "https://github.com/tulir/gomuks"

var (
	Version          string
	VersionDesc      string
	LinkifiedVersion string
	ParsedBuildTime  time.Time
)

var wantHelp, _ = flag.MakeHelpFlag()
var version = flag.MakeFull("v", "version", "View ingestor version and quit.", "false").Bool()

type BeeperIngestor struct {
	gmx         *gomuks.Gomuks
}

type Credentials struct {
	Username string
	Password string
}

func main() {
	hicli.InitialDeviceDisplayName = "gomuks web"
	initVersion(Tag, Commit, BuildTime)
	flag.SetHelpTitles(
		"ingestor - A thingie for getting messages from a gomuks instance.",
		"ingestor [-hv]",
	)
	err := flag.Parse()

	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		flag.PrintHelp()
		os.Exit(1)
	} else if *wantHelp {
		flag.PrintHelp()
		os.Exit(0)
	} else if *version {
		fmt.Println(VersionDesc)
		os.Exit(0)
	}

	gmx := gomuks.NewGomuks()
	gmx.Version = Version
	gmx.Commit = Commit
	gmx.LinkifiedVersion = LinkifiedVersion
	gmx.BuildTime = ParsedBuildTime
	gmx.FrontendFS = web.Frontend
	run(gmx)
}

func initDirectories(gmx *gomuks.Gomuks) {
	gomuksRoot := os.Getenv("GOMUKS_ROOT")
	if gomuksRoot == "" {
		panic("GOMUKS_ROOT environment variable is not set")
	}
	fmt.Println("GOMUKS_ROOT:", gomuksRoot)
	exerrors.PanicIfNotNil(os.MkdirAll(gomuksRoot, 0700))
	gmx.CacheDir = filepath.Join(gomuksRoot, "cache")
	gmx.ConfigDir = filepath.Join(gomuksRoot, "config")
	gmx.DataDir = filepath.Join(gomuksRoot, "data")
	gmx.LogDir = filepath.Join(gomuksRoot, "logs")

	if gmx.TempDir = os.Getenv("GOMUKS_TMPDIR"); gmx.TempDir == "" {
		gmx.TempDir = filepath.Join(gmx.CacheDir, "tmp")
	}
	exerrors.PanicIfNotNil(os.MkdirAll(gmx.ConfigDir, 0700))
	exerrors.PanicIfNotNil(os.MkdirAll(gmx.CacheDir, 0700))
	exerrors.PanicIfNotNil(os.MkdirAll(gmx.TempDir, 0700))
	exerrors.PanicIfNotNil(os.MkdirAll(gmx.DataDir, 0700))
	exerrors.PanicIfNotNil(os.MkdirAll(gmx.LogDir, 0700))

	configFilePath := filepath.Join(gmx.ConfigDir, "config.yaml")
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		panic(fmt.Sprintf("Config file does not exist: %s", configFilePath))
	}
	file, err := os.Open(configFilePath)
	if err != nil {
		panic(fmt.Sprintf("Failed to open config file: %s", err))
	}
	defer file.Close()
}

func run(gmx *gomuks.Gomuks) {
	initDirectories(gmx)
	err := gmx.LoadConfig()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to load config:", err)
		os.Exit(9)
	}
	gmx.SetupLog()
	gmx.Log.Info().
		Str("version", gmx.Version).
		Str("go_version", runtime.Version()).
		Time("built_at", gmx.BuildTime).
		Msg("Initializing gomuks")
	ab := &BeeperIngestor{
		gmx: gmx,
	}
	ab.StartServer()
	gmx.StartClient()
	gmx.Log.Info().Msg("Initialization complete")
	gmx.WaitForInterrupt()
	gmx.Log.Info().Msg("Shutting down...")
	gmx.DirectStop()
	gmx.Log.Info().Msg("Shutdown complete")
	os.Exit(0)
}

func (ab *BeeperIngestor) StartServer() {
	router := http.NewServeMux()
	router.HandleFunc("/search-messages", ab.SearchMessages)

	accessList := parseAccessList()
	handler := basicAuthMiddleware(accessList)(router)

	ab.gmx.Server = &http.Server{
		Addr:    ab.gmx.Config.Web.ListenAddress,
		Handler: handler,
	}
	go func() {
		err := ab.gmx.Server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	ab.gmx.Log.Info().Str("address", ab.gmx.Config.Web.ListenAddress).Msg("Server started")
}

func parseAccessList() map[string]string {
	accessList := make(map[string]string)
	rawList := os.Getenv("ACCESS_LIST")
	if rawList == "" {
		log.Fatal("ACCESS_LIST environment variable is required.")
	}

	pairs := strings.Split(rawList, "|")
	for _, pair := range pairs {
		creds := strings.Split(pair, ":")
		if len(creds) != 2 {
			log.Fatal("Invalid ACCESS_LIST format. Expected format: user:hashedpass|user2:hashedpass2")
		}
		// The password should already be hashed using SHA-256 + Base64 from generate-password.py
		username, hashedPass := creds[0], creds[1]

		// Basic validation of the hash format (SHA-256 in Base64 is always 44 characters)
		if len(hashedPass) != 44 {
			log.Fatalf("Invalid hash length for user %s.", username)
		}

		accessList[username] = hashedPass
	}
	return accessList
}

func basicAuthMiddleware(accessList map[string]string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			storedHash, exists := accessList[username]
			if !exists {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Hash the provided password using the same method as generate-password.py
			hasher := sha256.New()
			hasher.Write([]byte(password))
			passwordHash := base64.StdEncoding.EncodeToString(hasher.Sum(nil))

			if storedHash != passwordHash {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func initVersion(tag, commit, rawBuildTime string) {
	if len(tag) > 0 && tag[0] == 'v' {
		tag = tag[1:]
	}
	if tag != StaticVersion {
		suffix := "+dev"
		if len(commit) > 8 {
			Version = fmt.Sprintf("%s%s.%s", StaticVersion, suffix, commit[:8])
		} else {
			Version = fmt.Sprintf("%s%s.unknown", StaticVersion, suffix)
		}
	} else {
		Version = StaticVersion
	}

	LinkifiedVersion = fmt.Sprintf("v%s", Version)
	if tag == Version {
		LinkifiedVersion = fmt.Sprintf("[v%s](%s/releases/v%s)", Version, URL, tag)
	} else if len(commit) > 8 {
		LinkifiedVersion = strings.Replace(LinkifiedVersion, commit[:8], fmt.Sprintf("[%s](%s/commit/%s)", commit[:8], URL, commit), 1)
	}
	if rawBuildTime != "unknown" {
		ParsedBuildTime, _ = time.Parse(time.RFC3339, rawBuildTime)
	}
	var builtWith string
	if ParsedBuildTime.IsZero() {
		rawBuildTime = "unknown"
		builtWith = runtime.Version()
	} else {
		rawBuildTime = ParsedBuildTime.Format(time.RFC1123)
		builtWith = fmt.Sprintf("built at %s with %s", rawBuildTime, runtime.Version())
	}
	mautrix.DefaultUserAgent = fmt.Sprintf("ingestor/%s %s", Version, mautrix.DefaultUserAgent)
	VersionDesc = fmt.Sprintf("ingestor %s (%s)", Version, builtWith)
	BuildTime = rawBuildTime
}
