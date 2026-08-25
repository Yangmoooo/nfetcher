# Changelog

## [v0.1.0] - 2026-08-25

### Added

- Published the first GHCR container image for `linux/amd64`.
- Added a generic Compose example and `.env.example` configuration template.
- Added automatic tests and GHCR image publishing through GitHub Actions.
- Added configurable output directory through `NF_LIBRARY_DIR`.

### Changed

- The default container output directory is now `/nhentai`.
- Sensitive settings use `NF_`-prefixed environment names.
- Public documentation uses the GHCR image as the primary installation path.
- Komga is documented as an optional consumer profile rather than a core dependency.
- Search results are used directly to build the download plan; gallery detail requests are no longer required.
- Official `ComicInfo.xml` fields are preserved while `StoryArc` and `StoryArcNumber` are added.

### Fixed

- Prevented multibyte titles from exceeding the filesystem filename component limit.
- Canceled jobs now stop before retention cleanup and report cancellation correctly.
