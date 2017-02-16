# Guide to *ssed*

![Coverage](https://img.shields.io/badge/coverage-90%25-brightgreen.svg)

*bol* is powered by a library to perform *simple synchronization of encrypted documents* (*ssed*), refereed to as the filesystem (fs). The following is the working document for the idea and implementation of the *ssed* fs.

## Principles

In its essence, *ssed* is a file system composed of documents. A **document** is a list of entries. An **entry** is a map containing:

- *Entry* name which must be unique (text)
- *Document* which the entry belongs to (text)
- *Text* content of the entry (text). Can also be indicator of "ignore document" / "ignore entry" to perform pseudo-deletion.
- *Timestamp* of the creation time, used to sort for display (timestamp).
- *ModifiedTimestamp* which is the last modified time, used to sort for ignoring (timestamp).

Each entry is stored in a separate file. The fs stores an entry by writing a JSON encoding the entry components to `UUID.json` where `UUID` is a sha256 hash of the entry content. The entry JSON is encrypted using 256-bit AES-GCM and stored as a hex string.


## Compression and Encryption

All entries are encrypted when they are saved to disk as individual files. All of these files are then collapsed into a `.tar.bz2` archive. Upon use, this archive is de-compressed, and then stored in a temp directory. Files are decrypted only when they are read.

__Note__: I'm aware that compression after encryption makes the archive slightly larger then encryption after compression. The increased costs are not exorbitant. Instead of compressing ~1,000 entries from 1MB to ~50k, instead it will compress to ~200k. The reason that I made this choice is the following: decompressing ~1,000 entries takes 2+ seconds on a typical laptop. If a password is not required for opening the archive, then the decompression can be performed asynchronously, while the password is being entered. Thus, program startup becomes an almost imperceptible <200 ms instead of a tiresome 2+ seconds. Decryption is much faster, and can be performed on the documents once the password is entered.



## Synchronization

The local fs must *always* pull before it can push, because the local may be ahead or behind the remote. If the local fs is unable to pull, then it will avoid pushing during the session, in order not to overwrite newer entry on the remote.

Pulling is performed by unzipping the remote archive and local archive into the same directory and then rezipping them. Since all entries are stored as individual files, then if the local is ahead or behind, it will simply combine its file in.

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

### Synchronization methods

There are two possible methods for syncing.

#### Method 1 - Server (~500 ms upload/download)

Syncing is provided using a server and client. The server has two routes which the client can use:

- `GET /repo` - getting the latest archive, *not* protected, since it is encrypted
- `POST /repo` - pushing changes to an archive, requires basic authorization
- `PUT /repo` - add a user, requires basic authorization for credentials

#### Method 2 - SSH remote computer (~1500 ms upload/download) - not yet implemented

SSH is provided by the sftp library which can upload and download.

The user needs to provide:

- server address
- private SSH key OR password to access server

## Adding and viewing entries

Adding/viewing entries can be done using the command line program or the server (though in a more limited way).

## Other purposeful neglectfulness

After all, *simple* is part of *ssed*. In that light...

Diffs will not be stored. I'd like to optionally add an entry at any point in time without rebuilding history.

Files can not be deleted. It makes synchronization easier and also the disk cost of words is VERY small so its fine to store tons of text files (~1000's). If you plan on having 100,000+ entries, then this is not the tool for you.


## Configuration

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

## Exporting

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

## Testing

This library can be tested by first running a *bolserver* and then using `go test`. E.g.

```
cd ../bolserver && go build && ./bolserver
```

Then in another terminal do:

```
go test -coverprofile cover.out && go tool cover -html=cover.out -o index.html && python3 -m http.server
```
