FROM golang:latest
RUN useradd -M "mpdsub" -K UID_MIN=10000 -K GID_MIN=10000

WORKDIR /go/src/github.com/mdlayher/mpdsub
COPY . .

# required otherwise the user-ran build fails
RUN chown -R mpdsub /go/src/

USER "mpdsub"

RUN go-wrapper download   # "go get -d -v ./..."
RUN go-wrapper install    # "go install -v ./..."

EXPOSE 4040

#CMD ["go-wrapper", "run"]
ENTRYPOINT ["go", "run", "cmd/mpdsubd/main.go"]
