# Changelog

## [Unreleased]

## [v0.2.2] - 2026-11-20

### Changed

- With `NF_KOMGA_READ_LIST` enabled, rewrite only `ComicInfo.xml` and copy all other CBZ entries without recompression or per-entry CRC validation.
- Require Go 1.26.0 or later to build from source. CI and release images use Go 1.27.4.

## [v0.2.1] - 2026-08-29

### Changed

- Enabling `NF_KOMGA_READ_LIST` now strips ComicInfo `AlternateSeries`, `AlternateNumber`, and `AlternateCount` so Komga only creates date-based Read Lists.

## [v0.2.0] - 2026-08-27

### Changed

- Runtime configuration now fails fast when numeric, floating-point, or duration environment variables are malformed.
- Retention now uses CBZ file modification time instead of ComicInfo `StoryArc` metadata.
- Komga Read List metadata is optional through `NF_KOMGA_READ_LIST` and disabled by default. Komga users who want date-based Read Lists must set `NF_KOMGA_READ_LIST=true`.

### Removed

- Removed unused legacy storage and naming helpers.

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
