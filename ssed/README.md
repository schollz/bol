*bol* is powered by a library to perform simple synchronization of encrypted documents (`ssed`). This is refered to to as the `ssed` filesystem (fs). The following is the working document for the idea and implementation of the `ssed` fs.

# Guide to `ssed`

`ssed` is simple synchronization of encrypted documents. `ssed` stores documents. A document is a list of entries. A single entry has:

- name of entry, must be unique (text).
- name of document it belongs to (text)
- content of the entry (text). Can also be indicator of "ignore document" / "ignore entry" to perform pseudo-deletion.
- creation time, used to sort for display (timestamp).
- last modified time, used to sort for ignoring (timestamp).

Each entry is stored in a separate file. The fs stores an entry by writing a JSON encoding the entry components to `UUID.json` where `UUID` is a sha256 hash of the entry content. This entry is encrypted using AES and stored as a hex string.


# Compression / Encryption

All entries for all documents are stored in an a  tar.bz2 archive. Upon use, this archive is de-compressed, and then stored in a temp directory. Files are decrypted only when they are read.

__Note__: I'm aware that compressing after encryptiong makes the archive slightly larger (since it is compressing encrypted text). However, I've found that decompressing >1,000 files takes 1.5+ seconds.
Thus, I aim to perform decompression *asynchronously*, while the password is being entered, which means the archive itself cannot be encrypted.
The increased costs are not exorbitant. Instead of compressing 1000 short documents from 1MB to ~50k, instead it will compress to ~200k.


# Syncing

The local fs must *always* pull before it can push, because the local maybe ahead or behind. Pulling is performed by unzipping the remote archive and local archive into the same directory and then rezipping them. If the local is ahead or behind, it will simply combine its file in.

Synchronization occurs in two steps. First the user initializes the filesystem. Then the password is supplied.

```golang
var fs ssed
fs.Init(username,method)
fmt.Printf("Enter password: ")
var password string
fmt.Scanln(&password) // user types in password
err = fs.Open(password) // does not run until Init(..) is finished
if err == nil {
  // good password
} else {
  // incorrect password
}
```

The `Init(..)` function will download the latest repo for `username`, and merge local+remote contents *asynchronously*. These steps are also decoupled from requiring any passwords, so they will not need to wait for a password to be entered. In the meantime, the password can be requested and supplied.

The method `Open(..)` checks the password by trying to decrypt an entry, and if it fails it returns an error. The function, `Open(..)` will not start until the initialization is done, but the initialization will run while the user spends time typing in a password.

## Methods

There are two possible methods for syncing.

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

Adding entries should also done using the server. Viewing entries can *not* be done using the server to avoid having to have a front-end and front-end security.

## Other purposeful neglectfulness

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

Paths:
```
pathToLocalArchive:   $HOME/.cache/ssed/local/username.tar.bz2
pathToLocalEntries:   $HOME/.cache/ssed/local/username/
pathToRemoteArchive:  $HOME/.cache/ssed/remote/username.tar.bz2
pathToRemoteEntries:  $HOME/.cache/ssed/remote/username/
pathToTemp:           $HOME/.cache/ssed/temp
pathToConfigFile:     $HOME/.config/ssed/config.json
```

Only `pathToTemp` contains unencrypted things, so all files in that folder should be shredded upon exit.

# Exporting

Exporting releases a JSON file that is simply a list of documents (which are lists of entries associated with each document).

```javascript
[  
  {  
    "Name":"document1",
    "Entries":[  
      {  
        "text":"some text",
        "timestamp":"2014-11-20 13:00:00",
        "modified_timestamp":"2014-11-20 13:00:00",
        "document":"document1",
        "entry":"JwZGmuuykV"
      },
      {  
        "text":"some text4",
        "timestamp":"2013-11-20 13:00:00",
        "modified_timestamp":"2013-11-20 13:00:00",
        "document":"document1",
        "entry":"entry2"
      }
    ]
  }
]
```
