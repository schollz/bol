# bol

> I wanted a notebook that functioned not as a body but as a mind, a notebook that collected, interposed, collaged: a machine whose components could move, whose cogs, chutes, and levers were air. - [Patricia Lockwood](http://www.newyorker.com/magazine/2016/11/28/finding-poetry-in-a-note-taking-app)

*bol* is [a command-line program](https://github.com/schollz/bol/releases) that lets you write/view encrypted documents and [a webpage](https://bol.schollz.com/) that lets you write (not view) documents.

*bol* uses `ssed` as a backend for the encrypted storage and synchronization. For more information, [see the white paper](https://github.com/schollz/bol/blob/master/ssed/README.md).

## Install

```
go get -u -v github.com/schollz/bol/...
```

## Usage

Just run

```
bol
```

### Delete entry

To delete entry, just delete the entire entry and replace with ```ignore entry```.

### Delete document

To delete a document, just make a new entry that says ```ignore document```.

### Edit specific entry/document

```
bol document_name
bol entry_name
```

### Backup everything

```
bol --dump
```

The dump file `user-20ZZ-YY-XX.bol` can be read using the boltool,

```
boltool -decrypt user-20ZZ-YY-XX.bol
```

### Erase all local copies

```
bol --clean
```

### Change user

```
bol --config
```
