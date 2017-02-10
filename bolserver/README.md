# bolserver

The *bolserver* is the component that helps with syncing of the documents. It handles synchronization with POST and GET requests. It requires registering users. The password is used to verify authenticity so that repositories can be synchronized.

## Dev

To build *bolserver*, make sure to re-bundle the static assets:

```
go get -u github.com/jteeuwen/go-bindata/...
go-bindata static/ login.html post.html
go build -v
```
