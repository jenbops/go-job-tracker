FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /jobtracker

FROM gcr.io/distroless/static
COPY --from=build /jobtracker /jobtracker
ENV JOBTRACKER_DB=/data/jobtracker.db
ENTRYPOINT ["/jobtracker"]
CMD ["serve", "-addr", "0.0.0.0:8080"]

