[![Build Status](https://travis-ci.com/dokzlo13/getthecat.svg?branch=master)](https://travis-ci.com/dokzlo13/getthecat) [![Go Report Card](https://goreportcard.com/badge/github.com/dokzlo13/getthecat)](https://goreportcard.com/report/github.com/dokzlo13/getthecat)

# GetTheCat API ![](https://img.shields.io/badge/version-alpha--0.1-yellow.svg "version")
**Simple service for creating your own [random-cat](https://thecatapi.com/) API**

>Time spent with cats is never wasted.
> -- Sigmund Freud

![](https://raw.githubusercontent.com/dokzlo13/getthecat/master/imgs/cats.jpg "cats.jpg")

**Features:**
- Automatic images collection
- Different images-engine support:
   - [google custom search](https://developers.google.com/custom-search/v1/overview)
   - [flickr api](https://www.flickr.com/services/api/)
- Random and recently added images
- Images info
- Dynamic views counting
- [Sqlite](https://www.sqlite.org/) as main storage
- Caching:
    - [Redis](https://redis.io/)
    - Built in cache engine
- Different configurable endpoints (cats, parrots, etc...)
- Serverless
- Different serving modes:
    - Saving images
    - Redirecting to source

## Install
```shell
$ git clone https://github.com/dokzlo13/getthecat
$ cd ./getthecat
# Installing dependencies
$ dep ensure
$ go build
```
## Run
```shell
$ chmod +x ./getthecat
$ ./getthecat -conf configfile.yaml
```

## Configs
Config file separated by sections:
- **engine** - Engine for image collection, string, one of:
    - google
    - flickr
- **auth** - Configs for engine api auth

    **for google**:
    - apikey: apikey, string
    - cx: custom search id, string (_more info [here](https://developers.google.com/custom-search/v1/cse/list)_)
  
    **for flickr**:
    - apikey: apikey, string
- **endpoints** - Tags, to collect images also endpoints for getthecat API. string list whithout whitespaces.
- **folder** - Folder to storing collected images, string
- **db** - Database file, to store info, string
- **debug** - Debuglevel, int, 0-3
- **logfile** - File, to storing logs, string
- **watcher** - Image collector settings, has fields:
    - maximal_uses - How many views can have image, before _watcher_ will collect new one, int
    - minimal_aviable - how many images to store with views, less then "maximal_uses", int
    - checktime - time delay between database syncs and new images collectings
    - collect - which info about images need to be collected, string, one of:
        - urls (collect only web images links)
        - files (collect full files)
    - cache - optional section for Redis configs, has fields:
        - addr - redis-connection url, string
        - db - redis DB to use, int
- **server** - Server settings section, has fields:
    - mode - images serving mode, string, one of:
        - cache (send user image, stored on disc)
        - proxy (send user redirect to image original url)
    - apipath - url path, to access configured endpoints

## Example config
```yaml
engine: flickr
auth:
  apikey: "YOUR_FLICKR_APIKEY"
endpoints:
  - cat
  - parrot
watcher:
  minimal_aviable: 5
  maximal_uses: 1
  checktime: 15
  collect: files
server:
  mode: cache
  apipath: getthecat/v1
folder: ./images
db: ./images/getthecat.db
debug: 1
logfile: ./getthecatapi.log
```


### Endpoints:
After starting service, it will collect images from selected engine with tags, selected in config file. Now, you can access it with path:
```url
https://host:8080/*apipath*/*endpoint*
```
With config, described here it will be paths:
```url
https://host:8080/getthecat/v1/cat
https://host:8080/getthecat/v1/parrot
```
Here is some endpoints:
```
/info/new
/info/rand
/info/static/:id
/img/new
/img/rand
/img/static/:id
```
Endpoints, started with path "_info_" responding JSON with images info.
Endpoints, started with path "_img_" responding images. Also, send header X-IMAGE-ID, contains image id in system.
Endpoints, ending with "_/new_" responds new image, which has lowest amount of views.
Endpoints, ending with "_/rand_" responds random image from database.
Also, images info and files can be accessed by their ID by endpoint "_/static/:id_".

### Requests examples:
_The [httpie](https://httpie.org/) tool is used here_
**Getting random image info**
```
$ http GET http://127.0.0.1:8080/getthecat/v1/cat/info/rand
HTTP/1.1 200 OK
Content-Length: 255
Content-Type: application/json; charset=utf-8
Date: Wed, 05 Dec 2018 14:28:50 GMT
{
    "data": {
        "Checksum": "4762d4e60f7507b9114e222fc6c0529b",
        "Height": 768,
        "Origin": "https://farm3.staticflickr.com/2892/9147153988_b6194198e0_b.jpg",
        "Width": 1024,
        "filesize": 304435,
        "id": "5f74d0d9-0d5a-4d50-acdd-eed1fb4a18f7",
        "type": "cat",
        "watched": 4
    },
    "error": ""
}
```
**Getting static image by ID**
```
$ http GET http://127.0.0.1:8080/getthecat/v1/cat/img/static/5f74d0d9-0d5a-4d50-acdd-eed1fb4a18f7
HTTP/1.1 200 OK
Accept-Ranges: bytes
Content-Length: 304435
Content-Type: image/jpeg
Date: Wed, 05 Dec 2018 14:30:23 GMT
Last-Modified: Sat, 01 Dec 2018 18:41:12 GMT
+-----------------------------------------+
| NOTE: binary data not shown in terminal |
+-----------------------------------------+
```

**Getting static actual image**
```
$ http GET http://127.0.0.1:8080/getthecat/v1/cat/img/new
HTTP/1.1 200 OK
Accept-Ranges: bytes
Cache-Control: max-age=0 no-cache no-store must-revalidate
Content-Length: 141858
Content-Transfer-Encoding: binary
Content-Type: image/jpeg
Date: Wed, 05 Dec 2018 14:32:03 GMT
X-Image-Id: fe4a0ea0-e44a-4bec-b01e-c6067721224d
+-----------------------------------------+
| NOTE: binary data not shown in terminal |
+-----------------------------------------+
```
**If "proxy" mode used, api will response redirect:**
```
$ http GET http://192.168.10.202:8080/getthecat/v1/cat/img/rand 
HTTP/1.1 303 See Other
Content-Length: 91
Content-Type: text/html; charset=utf-8
Date: Wed, 05 Dec 2018 14:34:56 GMT
Location: https://farm5.staticflickr.com/4810/31081951047_920a479b54_b.jpg
<a href="https://farm5.staticflickr.com/4810/31081951047_920a479b54_b.jpg">See Other</a>.
```

