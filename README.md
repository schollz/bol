# edfs
encrypted distributed file system

# Guide

This filesystem (fs) stores documents. A document is composed of entries. A single entry has:

- text content, main data of entry
- timestamp, date of that entry
- document name which refers to which it belongs to
- entry name which is a unique identifier of this entry for a given document

The fs stores an entry by writing a JSON encoding the entry components to `UUID.aes` (AES encrypted) where `UUID` is random.

# Syncing

Syncing is provided using a server and client. The server has two routes which the client can use:

- `GET /` - getting the latest archive
- `POST /` - pushing changes

These two routes are protected by basic authentication. The basic authentication is determined on startup of server.

The syncing occurs by first getting the latest archive - the archive is downloaded via `GET`, then the archive is unzipped, and then the new files are copied over to the current working directory.

Syncing is finished by pushing the latest version - the local files are archived, and then pushed to the server via `POST`. The server then unzips the new archive and the old archive, and copies new files over to the old archive, and then creates a new archive based on those files which is available for future `GET`.
