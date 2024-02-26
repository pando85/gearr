# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.4](https://github.com/pando85/gearr/tree/v0.1.4) - 2024-02-26

### Added

* Add sonarr integration
* Change encode FFMPEG CRF from 28 to 21
* Add max size flag to sonarr episodes

### Build

* Update dependency @types/react to v18.2.58

### Fixed

* Remove unnecessary colon in log message
* Render short path max 20 chars
* Continue even when MovieFile or MediaInfo is nil

### Refactor

* Add common code to a library

## [v0.1.3](https://github.com/pando85/gearr/tree/v0.1.3) - 2024-02-23

### Added

* Add stale bot

### Build

* Update dependency sass to v1.71.1
* Update dependency react-virtualized-auto-sizer to v1.0.23
* Update dependency @types/node to v20.11.20

### Fixed

* Exclude hevc video codec
* Handle rate equal to zero case
* Correct way of define video profile in ffmpeg
* Update favicon with logo

## [v0.1.2](https://github.com/pando85/gearr/tree/v0.1.2) - 2024-02-20

### Build

* Downgrade typescript to v4

## [v0.1.1](https://github.com/pando85/gearr/tree/v0.1.1) - 2024-02-20

### Added

- Change default encode to x265 10bit
- Add renovate

### Build

- Update github.com/isayme/go-amqp-reconnect digest to fc811b
- Update wagoid/commitlint-github-action action to v5
- Update dependency web-vitals to v3
- Update dependency @types/node to v20.11.19
- Update module github.com/streadway/amqp to v1
- Update actions/checkout action to v4
- Update actions/setup-go action to v5
- Update module github.com/spf13/pflag to v1.0.5
- Update module github.com/avast/retry-go to v2.7.0+incompatible
- Update module gopkg.in/vansante/go-ffprobe.v2 to v2.1.1
- Update module github.com/google/uuid to v1.6.0
- Update module github.com/jedib0t/go-pretty/v6 to v6.5.4
- Update module github.com/sirupsen/logrus to v1.9.3
- Update module github.com/lib/pq to v1.10.9
- Update ubuntu:22.04 Docker digest to f9d633f
- Update module github.com/spf13/viper to v1.18.2
- Update dependency axios to v1.6.7
- Update dependency react-bootstrap to v2.10.1
- Update dependency react-virtualized-auto-sizer to v1.0.22
- Update material-ui monorepo to v5.15.10
- Update react monorepo
- Update dependency react-router-dom to v6.22.1
- Update dependency react-use-websocket to v4.7.0
- Update dependency sass to v1.71.0
- Update dependency @testing-library/react to v14
- Update dependency typescript to v5
- Update dependency @testing-library/user-event to v14

### Documentation

- Add contributors

### Refactor

- Remove old binary resources

## [v0.1.0](https://github.com/pando85/gearr/tree/v0.1.0) - 2024-02-08

### Added

- Sort by column

## [v0.0.1](https://github.com/pando85/gearr/tree/v0.0.1) - 2024-02-08

### Fixed

- Replace details dropdown with bootrstrap card

## [v0.0.0](https://github.com/pando85/gearr/tree/v0.0.0) - 2024-02-07

Initial release
