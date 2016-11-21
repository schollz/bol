# jot2

*jot2* is a command-line program that lets you write/view encrypted documents and a webpage that lets you write (not view) documents.

# Guide to `ssed`

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

The remote *always* as the latest. The local maybe ahead or behind. This means the local must always combine the archives by unzipping them into the same directory and then rezipping them. If the local is ahead or behind, it will simply combine its file in.

## Methods
There are two possible methods for syncing.

Each method proceeds by first downloading an archive of all the entries and unzipping it into a working directory. Then it unzips the current archvie intot he working directory, and then re-archives all those files as the current archive. *This way the local copies can never be overwritten*

If the download was successful, then, after writing, it is uploaded back to the remote. *It only uploads if download was successful*, because otherwise the archive can contain things out of sync.

## Method 1 - Server (~500 ms upload/download)

Syncing is provided using a server and client. The server has two routes which the client can use:

- `GET /get` - getting the latest archive
- `POST /post` - pushing changes

These two routes are protected by basic authentication. The basic authentication is determined on startup of server.

The user needs to provide:

- server address
- username and password

## Method 2 - SSH remote computer (~1500 ms upload/download)

SSH is provided by the sftp library which can upload and download.

The user needs to provide:

- server address
- private SSH key OR password to access server

# Adding/viewing entries

Adding/viewing entries can be done using the command line program.

Adding entries should also done using the server, using [trix](https://trix-editor.org/) and then a form for document name, user name, and password

# Purposeful neglectfulness

- I purposely do not want to store diffs, since I'd like to optionally add an entry at any point in time without rebuilding (rebasing) history.

# API 

These are the commands available to the user:

- Open(method): where method is "ssh://server" or "http://server". This tells the server to attempt to pull. and create files if nessecary. The NAMEOFREPO is the base58 encoding of the method.
- Close(): closes the repo, telling it to push. Though it only pushes if it was successful pulling
- Update(text,documentName,entryName,date): make a new entry (or edit old one if entryName is not empty). date can be empty, it will fill in the current date if so
- DeleteDocument(documentName): will simply Update("ignore-document",documentName,"","")
- DeleteEntry(documentName,entryName): will simply Update("ignore-entry",documentName,entryName,"")
- GetEntry(documentName,entryName): returns all versions of entry, ordered by date
- GetDocument(documentName): returns latest versions of all entries in document, ordered by date 

## Implementation notes

Method 1 and 2 stores files on server as `$HOME/.cache/ssed/server/NAMEOFREPO.tar.bz2`. 
Local stores files as `$HOME/.cache/ssed/local/NAMEOFREPO/NAMEOFREPO.tar.bz2` and the temp files (for unziping are stored in) `$HOME/.cache/ssed/local/NAMEOFREPO/temp`. Basically the working directory is simply `$HOME/.cache/ssed/local/NAMEOFREPO/`.

Passwords for the archive are never stored. Authentication for accessing the server (name/password for Method 1 or password for Method 2) are stored in an encrypted document `$HOME/.cache/ssed/local/NAMEOFREPO/config.json`.
