FROM golang:1.15-alpine
WORKDIR /var/app
COPY . .
RUN apk add gcc g++ wget tesseract-ocr-dev
RUN wget https://github.com/tesseract-ocr/tessdata/raw/master/eng.traineddata
RUN go build -o app

FROM alpine
WORKDIR /var/app
COPY --from=0 /var/app .
RUN apk add ca-certificates tesseract-ocr
RUN mv eng.traineddata /usr/share/tessdata/
ENTRYPOINT ./app
