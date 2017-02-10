# :book: :palm_tree: bol

> I wanted a notebook that functioned not as a body but as a mind, a notebook that collected, interposed, collaged: a machine whose components could move, whose cogs, chutes, and levers were air. - [Patricia Lockwood](http://www.newyorker.com/magazine/2016/11/28/finding-poetry-in-a-note-taking-app)

*bol* is a system for editing and synchronization of encrypted documents. The main utility is a command-line program that lets you write/view encrypted documents. Synchronization is provided through a *bolserver* which handles synchronization and also allows writing new entires to documents. Both utilities are available in [the latest release](https://github.com/schollz/bol/releases/latest) as a self-contained executable binary.

# Install

[Download from releases](https://github.com/schollz/bol/releases/latest).

Or, if you have Go1.7+, just install via `go get`:

```
go get -u -v github.com/schollz/bol/...
```

# Usage

Just run

```
bol
```


**Delete an entry** by editing the entry and replacing the entire entry with ```ignore entry```. **Delete a document** by making a new entry in that document that says ```ignore document```. However, nothing is really deleted, as *bol* will save a copy of every entry and change committed to it.

**Edit a specific document/entry** using the command `bol DocumentName/EntryName`.

**Backup everything** to an AES-encrypted JSON file using `bol -dump`. The dump file `user-20ZZ-YY-XX.bol` can be read using the boltool, `boltool -decrypt user-20ZZ-YY-XX.bol`.

**Erase all local files** using `bol -clean`. This will not remove remote files, there is no way to remove remote files.

**Change user** or the server, by using `bol -config`.

# Server

The default server is a public server, https://bol.schollz.com. You can also run your own server simply running `bolserver`. Then, use `bol -config` and type in the server address, now `http://localhost:9095` or whichever you set it to.

The server is optional, and only required if you want synchronized entires on multiple computers. *bol* will function locally with or without Internet. The server is useful if you wish to edit your document on multiple computers, and also will allow you to add new entries via the website.

# Implementation

*bol* uses `ssed` as a backend for the encrypted storage and synchronization. For more information, [see the `ssed` README](https://github.com/schollz/bol/blob/master/ssed/README.md). Essentially, *bol* stores all entries as AES-encrypted JSON files which are then bundled as a `tar.bz2` archive which is synchronized with a server. Nothing can ever be deleted (only ignored), and all changes are stored.

# Contributing

I'm open to contributions, submit a pull request.

# License

MIT
