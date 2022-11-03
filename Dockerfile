FROM library/golang:1.19.3-bullseye AS backendbuild

WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download
COPY ./ /src/

RUN go build -o xp-state-metrics

#FROM ...distroless

#ENTRYPOINT ["/opt/app/xp-state-metrics"]
#WORKDIR /opt/app

#COPY --from=backendbuild /src/xp-state-metrics .
