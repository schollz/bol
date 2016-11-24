*bol* is powered by `ssed` filesystem (fs).

# Guide to `ssed`

`ssed` is simple synchronization of encrypted documents. `ssed` stores documents. A document is composed of entries. A single entry has:

- text content, main data of entry
- timestamp, date of that entry
- document name which refers to which it belongs to
- entry name which is a unique identifier of this entry for a given document

The fs stores an entry by writing a JSON encoding the entry components to `UUID.json` where `UUID` is a sha256 hash of the text. This entry is encrypted using AES.

The timestamp is used to provide the ordering.
The document is used to filter out only the needed entries.
The text content is either the fulltext of that entry, or an indicator ("ignore document" / "ignore entry") to help with reconstruction.

# Compression / Encryption

The archive is a  tar.bz2 archive of all entries. Upon use, this archive is de-compressed, and then stored in a temp directory. Files are decrypted only when they are read.

I'm aware that this makes the archive slightly larger (since it is compressing encrypted text). However, I've found that decompressing >1,000 files takes 1.5+ seconds.
Thus, I aim to perform decompression *asynchronously*, while the password is being entered, which means the archive itself cannot be encrypted.
Actual costs are not exorbitantly. Instead of compressing 1000 short documents from 1MB to ~50k, instead it will compress to ~200k.


# Syncing

The remote *always* has the latest. The local maybe ahead or behind. This means the local must always combine the archives by unzipping them into the same directory and then rezipping them. If the local is ahead or behind, it will simply combine its file in.

Synchronization occurs in two steps. First the user initializes the filesystem:

```golang
var fs ssed
fs.Init(username,method)
```

This will download the latest repo, and unzip both of them and merge there contents *asynchronously*. These steps are also decoupled from requiring any passwords, so they will not need to wait for a password to be entered. In the meantime, the password can be requested

```golang
for {
  password = AskUser() // 1 - 3 seconds
  err := fs.Open(password)
  if err == nil {
    break
  } else {
    fmt.Println("Incorrect password")
  }
}
```

During this stage, the password is requested, while the initialization is happening. Then the repository can be opened, given the right password, however `Open(..)` will not start until the initialization is done.

The method `Open(..)` checks the password by trying to decrypt an entry, and if it fails it returns an error.

## Methods
There are two possible methods for syncing.

Each method proceeds by first downloading an archive of all the entries and unzipping it into a working directory. Then it unzips the current archvie intot he working directory, and then re-archives all those files as the current archive. *This way the local copies can never be overwritten*

If the download was successful, then, after writing, it is uploaded back to the remote. *It only uploads if download was successful*, because otherwise the archive can contain things out of sync.

## Method 1 - Server (~500 ms upload/download)

Syncing is provided using a server and client. The server has two routes which the client can use:

- `GET /get` - getting the latest archive, *not* protected, since it is encrypted
- `POST /post` - pushing changes to an archive, protected by basic auth (username + password is sent)
- `PUT /user` - add a user (username + hashed password is sent)

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


# Configuration

Whenever `Init(name, method)` is called, [it generates a JSON](https://play.golang.org/p/6jHI-MRx0z) containing the configuration file in `$HOME/.config/ssed/config.json:

```javascript
[  
   {  
      "username":"zack",
      "method":"ssh://server"
   },
   {  
      "username":"bob",
      "method":"http://server2"
   },
   {  
      "username":"jill",
      "method":"http://server3"
   }
]
```

The first element of the configuration array is the default. Anytime a different configuration is used, it is popped from the list and added to the front and then resaved. The `username` is the name of the repo and the `method` is how to connect to the remote.

## Implementation notes

Method 1 and 2 stores files on server as `$HOME/.cache/ssed/username.tar.bz2`.
Local stores files as `$HOME/.cache/ssed/username.tar.bz2` and the temp files (for unzipping are stored in) `$HOME/.cache/ssed/temp`.

Before unzipping, the archive is moved to `$HOME/.cache/ssed/temp/data.tar.bz2` which is unzipped to `$HOME/.cache/ssed/temp/data`.
