FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS build
ARG GIT_VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/embed.go web/
COPY --from=web /src/web/dist web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${GIT_VERSION}" -o /comics ./cmd/comics

FROM alpine:3
ARG GIT_VERSION
ARG BUILD_DATE
LABEL org.label-schema.vcs-ref=$GIT_VERSION \
      org.label-schema.vcs-url="https://github.com/Oshuma/comics" \
      org.label-schema.build-date=$BUILD_DATE
RUN apk add --no-cache ca-certificates tzdata && adduser -D -h /data comics
COPY --from=build /comics /usr/local/bin/comics
USER comics
WORKDIR /data
EXPOSE 3000/tcp
CMD ["comics"]
