# igor-check-certs

## Usage

    $ go run check-certs.go -days 100 igor.io gif.industries expired.badssl.com
    error: expired.badssl.com: tls dial: x509: certificate has expired or is not yet valid
    error: gif.industries: cert[0] gif.industries expires at 2016-12-10 11:00:00 +0000 UTC
    error: igor.io: cert[0] igor.io expires at 2016-12-10 11:00:00 +0000 UTC
    exit status 1
