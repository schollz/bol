# Guide to `ssed`

`ssed` is simple synchronization of encrypted documents. `ssed` stores documents. A document is composed of entries. A single entry has:

- text content, main data of entry
- timestamp, date of that entry
- document name which refers to which it belongs to
- entry name which is a unique identifier of this entry for a given document

The fs stores an entry by writing a JSON encoding the entry components to `UUID.aes` (AES encrypted) where `UUID` is random.

The timestamp is used to provide the ordering.
The document is used to filter out only the needed entries.
The text content is either the fulltext of that entry, or an indicator ("ignore document" / "ignore entry") to help with reconstruction.

# Compression / Encryption

The archive is a  tar.bz2 archive.
Each document in the archive is encrypted in AES.
Upon use, this archive is de-compressed, and then stored in a temp directory. Files are decrypted only when they are read.

I'm aware that this makes the archive slightly larger (since it is compressing encrypted text). However, I've found that decompressing >1,000 files takes 1.5+ seconds.
Thus, I aim to perform decompression *asynchronously*, while the password is being entered, which means the archive itself cannot be encrypted.
Actual costs are not exorbitantly. Instead of compressing 1000 short documents from 1MB to ~50k, instead it will compress to ~200k.


# Syncing

The remote *always* has the latest. The local maybe ahead or behind. This means the local must always combine the archives by unzipping them into the same directory and then rezipping them. If the local is ahead or behind, it will simply combine its file in.

## Methods
There are two possible methods for syncing.

Each method proceeds by first downloading an archive of all the entries and unzipping it into a working directory. Then it unzips the current archvie intot he working directory, and then re-archives all those files as the current archive. *This way the local copies can never be overwritten*

If the download was successful, then, after writing, it is uploaded back to the remote. *It only uploads if download was successful*, because otherwise the archive can contain things out of sync.

## Method 1 - Server (~500 ms upload/download)

Syncing is provided using a server and client. The server has two routes which the client can use:

- `GET /get` - getting the latest archive, protected by basic auth
- `POST /post` - pushing changes to an archive, protected by basic auth
- `PUT /user` - add a user (username + hashed password is given to the server)

## Method 2 - SSH remote computer (~1500 ms upload/download)

SSH is provided by the sftp library which can upload and download.

The user needs to provide:

- server address
- private SSH key OR password to access server

# Adding/viewing entries

Adding/viewing entries can be done using the command line program.

Adding entries should also done using the server, using [trix](https://trix-editor.org/) or [quill](http://codepen.io/anon/pen/JbWvyY?editors=1111) and then a form for document name, user name, and password

# Purposeful neglectfulness

- Diffs will not be stored. I'd like to optionally add an entry at any point in time without rebuilding (rebasing) history.
- Files can not be deleted. It makes synchronization easier and also the disk cost of words is VERY small so its fine to store tons of text files (~1000's)

# API

These are the commands available to the user:

- `ListConfigs()`: method to load the configuration file and list the available configurations
- `Open(name,password,method)`: where method is "ssh://server" or "http://server". This tells the server to attempt to pull. and create files if nessecary. The name is the name to store the repo under. The password is used to unlock the repo as well as authenticate access to the HTTP server (SSH uses private key).
- `Close()`: closes the repo, telling it to push. Though it only pushes if it was successful pulling
- `Update(text,documentName,entryName,date)`: make a new entry (or edit old one if entryName is not empty). date can be empty, it will fill in the current date if so
- `DeleteDocument(documentName)`: will simply Update("ignore-document",documentName,"","")
- `DeleteEntry(documentName,entryName)`: will simply Update("ignore-entry",documentName,entryName,"")
- `GetEntry(documentName,entryName)`: returns all versions of entry, ordered by date
- `GetDocument(documentName)`: returns latest versions of all entries in document, ordered by date

Every action other than `Open()` and `Close()` will untar and decompress the archive, get the contents, and then retar it and compress (so it mostly stays in that state. Unless this is slow, then it will be open to Open and Close to do this.

# Configuration

Whenever `Open(..)` is called, [it generates a JSON](https://play.golang.org/p/6jHI-MRx0z) containing the configuration file in `$HOME/.config/ssed/config.json:

```javascript
[  
   {  
      "username":"zack",
      "password":"$2a$14$zKUKoJJv2tu8By5qcidfSO8oe/evLSYKQIhZdqBorWSSzYVb/.wuW",
      "method":"ssh://server"
   },
   {  
      "username":"bob",
      "password":"$2jlkjlkja9sdfawj3flkjw3fa/3wfkwjIhZdqBorWSSzYVb/.wuW",
      "method":"http://server2"
   },
   {  
      "username":"jill",
      "password":"23423#kjlkja9sdfawj3flkjw3fa/3wfkwjIhZdqBorWSSzYVb/.wuW",
      "method":"http://server3"
   }
]
```

The first element of the configuration array is the default. Anytime a different configuration is used, it is popped from the list and added to the front and then resaved. The `username` is the name of the repo, the `password` is the [hashed password](https://github.com/gtank/cryptopasta/blob/master/hash.go#L41) and the `method` is how to connect to the remote.

## Implementation notes

Method 1 and 2 stores files on server as `$HOME/.cache/ssed/server/username.tar.bz2`.
Local stores files as `$HOME/.cache/ssed/local/username.tar.bz2` and the temp files (for unziping are stored in) `$HOME/.cache/ssed/temp`.
