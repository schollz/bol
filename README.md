# ssed
simple synchronization and encryption of documents

# Guide

`ssed` stores documents. A document is composed of entries. A single entry has:

- text content, main data of entry
- timestamp, date of that entry
- document name which refers to which it belongs to
- entry name which is a unique identifier of this entry for a given document

The fs stores an entry by writing a JSON encoding the entry components to `UUID.aes` (AES encrypted) where `UUID` is random.

The timestamp is used to provide the ordering.
The document is used to filter out only the needed entries.
The text content is either the fulltext of that entry, or an indicator ("ignore document" / "ignore entry") to help with reconstruction.

# Compression / Encryption

The archive is a AES encrypted tar.bz2 archive. Upon use, this archive is decrypted, then decompressed, and then stored in a temp directory. When finished, the temp directory is archived and then encrypted and then shredded.

# Syncing

There are two possible methods.

Each method proceeds by first downloading an archive of all the entries and unzipping it into a working directory. If the download was successful, then, after writing, it is uploaded back to the remote. *It only uploads if download was successful*, because otherwise the archive can contain things out of sync.

## Method 1 - Server (~500 ms upload/download)

Syncing is provided using a server and client. The server has two routes which the client can use:

- `GET /` - getting the latest archive
- `POST /` - pushing changes

These two routes are protected by basic authentication. The basic authentication is determined on startup of server.

The user needs to provide:

- server address
- username and password

## Method 2 - SSH remote computer (~1500 ms upload/download)

SSH is provided by the sftp library which can upload and download.

The user needs to provide:

- server address
- private SSH key OR password to access server

# Purposeful neglectfulness

I purposely do not want to store diffs, since I'd like to optionally add an entry at any point in time without rebuilding (rebasing) history.
