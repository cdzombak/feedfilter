![feedfilter](feedfilter.png)

**feedfilter** allows filtering an RSS feed based on a [CEL](https://cel.dev) expression that decides which items to include.

## Usage

```bash
feedfilter [options]
```

### Options

- `-config <path>`: Path to config file (default: `./config.json`)
- `-version`: Print version and exit

### Configuration

feedfilter uses a JSON configuration file to specify the feed source, output destination, filtering rules, and output format. The configuration file should contain:

```json
{
  "from": "https://example.com/feed.xml",
  "to": "/path/to/output.xml",
  "to_fmt": "rss",
  "include_if": "title.contains('keyword') || categories.exists(c, c == 'tech')",
  "meta": {
    "title": "Filtered Feed",
    "description": "A filtered version of the original feed",
    "link": "https://example.com"
  }
}
```

To output to stdout instead of a file, set `"to": "-"`:

```json
{
  "from": "https://example.com/feed.xml",
  "to": "-",
  "to_fmt": "json",
  "include_if": "title.contains('keyword')"
}
```

#### Configuration Fields

- `from` (required): URL of the RSS/Atom feed to filter
- `to` (required): File path where the filtered feed will be written, or `"-"` to output to stdout
- `to_fmt` (required): Output format - `"rss"`, `"atom"`, or `"json"`
- `include_if` (optional): CEL expression to determine which items to include. If empty, all items are included.
- `meta` (optional): Metadata for the output feed
  - `title`: Feed title (use `"$$ORIG$$"` to use original feed's title)
  - `description`: Feed description (use `"$$ORIG$$"` to use original feed's description)
  - `link`: Feed link URL

#### CEL Expression Variables

The `include_if` CEL expression has access to these variables for each feed item:

- `title` (string): Item title
- `description` (string): Item description
- `link` (string): Item link
- `categories` (list of strings): Item categories

## Installation

### macOS via Homebrew

```shell
brew install cdzombak/oss/feedfilter
```

### Debian via apt repository

[Install my Debian repository](https://www.dzombak.com/blog/2025/06/updated-instructions-for-installing-my-debian-package-repositories/) if you haven't already:

```shell
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://dist.cdzombak.net/keys/dist-cdzombak-net.gpg -o /etc/apt/keyrings/dist-cdzombak-net.gpg
sudo chmod 644 /etc/apt/keyrings/dist-cdzombak-net.gpg
sudo mkdir -p /etc/apt/sources.list.d
sudo curl -fsSL https://dist.cdzombak.net/cdzombak-oss.sources -o /etc/apt/sources.list.d/cdzombak-oss.sources
sudo chmod 644 /etc/apt/sources.list.d/cdzombak-oss.sources
sudo apt update
```

Then install `feedfilter` via `apt-get`:

```shell
sudo apt-get install feedfilter
```

### Manual installation from build artifacts

Pre-built binaries for Linux and macOS on various architectures are downloadable from each [GitHub Release](https://github.com/cdzombak/feedfilter/releases). Debian packages for each release are available as well.

### Build and install locally

```shell
git clone https://github.com/cdzombak/feedfilter.git
cd feedfilter
make build
cp out/feedfilter $INSTALL_DIR
```

## Docker images

Docker images are available for a variety of Linux architectures from [Docker Hub](https://hub.docker.com/r/cdzombak/feedfilter) and [GHCR](https://github.com/cdzombak/feedfilter/pkgs/container/feedfilter). Images are based on the `scratch` image and are as small as possible.

Run them via, for example:

```shell
docker run -v ./config.json:/config.json --rm cdzombak/feedfilter:1 -config /config.json
docker run -v ./config.json:/config.json --rm ghcr.io/cdzombak/feedfilter:1 -config /config.json
```

## License

GNU General Public License v3.0; see [LICENSE](LICENSE) in this repository.

## Author

This project is maintained by Chris Dzombak ([dzombak.com](https://www.dzombak.com) / [github.com/cdzombak](https://www.github.com/cdzombak)).
