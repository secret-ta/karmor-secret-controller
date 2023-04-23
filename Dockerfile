FROM golang:1.18.8-alpine3.16 AS BuildStage

WORKDIR /app

COPY . .

RUN rm -rf example Dockerfile install.yaml

RUN go mod download

RUN go build -o /karmor-secret-controller .


FROM alpine:latest

WORKDIR /

COPY --from=BuildStage /karmor-secret-controller /karmor-secret-controller

ENTRYPOINT ["/karmor-secret-controller"]
