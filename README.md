# dmachat/dataurltopng

dataurltopng provides a service to convert base64 encoded png strings from html5 dataurl attributes into real png files

## Usage

Compile the binary and run on your server, optionally using a Config.gcfg file to overwrite basic auth and image path values.

POST requests should look like

```json
{
  Sitename: "sitename",
  Dataurl: "data:image/png;base64,..."
}
```

## Configuration

You can add a go config file to set environment variables like port, authentication information and images directory:

```ini
[server]
port = "9000"
imagedir = "/usr/local/images"
username = "data"
password = "topng"
```

You'll get back a json object with a success and filepath property.

## License

MIT
