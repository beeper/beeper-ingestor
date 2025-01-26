FROM golang:1.23 AS builder

RUN apt-get update && apt-get install -y libolm-dev libolm3

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download
RUN go install golang.org/x/tools/cmd/goimports@latest
RUN go install honnef.co/go/tools/cmd/staticcheck@latest

COPY . .

ENV PATH="/root/go/bin:${PATH}"

RUN export MAUTRIX_VERSION=$(cat go.mod | grep 'maunium.net/go/mautrix ' | head -n1 | awk '{ print $2 }')

RUN go build -ldflags "-X main.Tag=unknown -X main.Commit=unknown -X 'main.BuildTime=$(date -Iseconds)' -X 'maunium.net/go/mautrix.GoModVersion=$MAUTRIX_VERSION'" -o ingestor ./cmd/ingestor

# FROM dock.mau.dev/tulir/gomuks:webmuks

# RUN apk --no-cache add ca-certificates libc6-compat libolm-dev libolm3

# WORKDIR /ingestor-app

# COPY --from=builder /app/ingestor .

# COPY --from=builder /app/config ./config

EXPOSE 29325

# CMD ["/ingestor-app/ingestor"]

CMD ["/app/ingestor"]