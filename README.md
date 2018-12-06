## [![Build Status](https://travis-ci.com/dokzlo13/getthecat.svg?branch=master)](https://travis-ci.com/dokzlo13/getthecat) [![Go Report Card](https://goreportcard.com/badge/github.com/dokzlo13/getthecat)](https://goreportcard.com/report/github.com/dokzlo13/getthecat)

# GetTheCat API ![](https://img.shields.io/badge/version-alpha--0.1-yellow.svg "version")
**Simple service for creating your own [random-cat](https://thecatapi.com/) API**

>Time spent with cats is never wasted.
> -- Sigmund Freud

![](https://raw.githubusercontent.com/dokzlo13/getthecat/master/imgs/cats.jpg "cats.jpg")

## **Features:**
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
- High prefomance (kind of)
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
$ # as default, programm will use "config.yaml" in current directory, also supports yaml|toml|json|xml formats
$ ./getthecat -conf /path/to/configfile.yaml
```

## Configuring
Config file separated by sections:
- **engine** - Engine for image collection, string, one of:
    - google
    - flickr
- **auth** - Configs for engine api auth
    - **for google**:
        - apikey: apikey, string
        - cx: custom search id, string (_more info [here](https://developers.google.com/custom-search/v1/cse/list)_)
    - **for flickr**:
        - apikey: apikey, string
- **endpoints** - Tags, to collect images also endpoints for getthecat API. string list whithout whitespaces.
- **folder** - Folder to storing collected images, string
- **db** - Database file, to store info, string
- **debug** - Debuglevel, int, 0-3
- **logfile** - File, to storing logs, string
- **watcher** - Image collector settings, has fields:
    - **maximal_uses** - How many views can have image, before _watcher_ will collect new one, int
    - **minimal_aviable** - how many images to store with views, less then "maximal_uses", int
    - **checktime** - time delay between database syncs and new images collectings, int, minutes
    - **collect** - which info about images need to be collected, string, one of:
        - **urls** (collect only web images links)
        - **files** (collect full files)
    - **cache** - optional section for Redis configs, has fields:
        - **addr** - redis-connection url, string
        - **db** - redis DB to use, int
- **server** - Server settings section, has fields:
    - **mode** - images serving mode, string, one of:
        - **cache** (send user image, stored on disc)
        - **proxy** (send user redirect to image original url)
    - **apipath** - url path, to access configured endpoints

## Example config
```yaml
engine: flickr
auth:
  apikey: "YOUR_FLICKR_APIKEY"
  
#Or like this:
#engine: google
#auth:
#  apikey: "GOOGLE_APIKEY"
#  cx: "CX_KEY"

endpoints:
  - cat
  - parrot
  
watcher:
  minimal_aviable: 5
  maximal_uses: 1
  checktime: 15
  collect: files
  # This is optional!
  cache:
    addr: localhost:6379
    db: 0

server:
  mode: cache
  apipath: getthecat/v1
  
folder: ./images
db: ./images/getthecat.db
debug: 1
logfile: ./getthecatapi.log
```


## Endpoints:
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


| Endpoint             | Action                                      | 
|:---------------------|:-------------------------------------------:|
| GET /info/new        | Actual (hass less views) image info in JSON | 
| GET /info/rand       | Random image info in JSON                   | 
| GET /info/static/:id | Image info by image ID                      | 
| GET /img/new         | Actual (hass less views) image file         | 
| GET /img/rand        | Random image file                           | 
| GET /img/static/:id  | Image info by image ID                      | 

Methods /img/new, /img/rand send header X-IMAGE-ID, contains image id.



## Cases examples:
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
        "Origin": "https://farm3.staticflickr.co....",
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

**Getting actual image**
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

## Performance
Benchmarks was running on Intel(R) Pentium(R) CPU 2117U @ 1.80GHz, 4GB RAM, HDD.


**GoBench:**
```$xslt
BenchmarkMemCache_SetCache-2       	  500000	      5931 ns/op	     983 B/op	       2 allocs/op
BenchmarkMemCache_GetAviable-2     	   10000	    368144 ns/op	     160 B/op	       2 allocs/op
BenchmarkRedisCache_Set-2          	    5000	    237442 ns/op	    1416 B/op	      24 allocs/op
BenchmarkRedisCache_GetAviable-2   	   10000	    141705 ns/op	     688 B/op	      17 allocs/op
PASS
ok  	getthecat	14.542s

```


**Perfomance tests with "[hey](https://github.com/rakyll/hey)" app.**

**Redis enabled:**
```
$ hey -n 3000 -c 300 http://127.0.0.1:8080/getthecat/v1/cat/img/rand

Summary:
  Total:	3.8375 secs
  Slowest:	0.7835 secs
  Fastest:	0.0016 secs
  Average:	0.3681 secs
  Requests/sec:	781.7567
  
  Total data:	641385067 bytes
  Size/request:	213795 bytes

Response time histogram:
  0.002 [1]	|
  0.080 [181]	|■■■■■■■■
  0.158 [69]	|■■■
  0.236 [213]	|■■■■■■■■■■
  0.314 [366]	|■■■■■■■■■■■■■■■■■
  0.393 [883]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.471 [662]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.549 [282]	|■■■■■■■■■■■■■
  0.627 [261]	|■■■■■■■■■■■■
  0.705 [75]	|■■■
  0.783 [7]	|


Latency distribution:
  10% in 0.1716 secs
  25% in 0.3023 secs
  50% in 0.3739 secs
  75% in 0.4464 secs
  90% in 0.5565 secs
  95% in 0.5958 secs
  99% in 0.6824 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0089 secs, 0.0016 secs, 0.7835 secs
  DNS-lookup:	0.0000 secs, 0.0000 secs, 0.0000 secs
  req write:	0.0008 secs, 0.0000 secs, 0.0236 secs
  resp wait:	0.3060 secs, 0.0013 secs, 0.6491 secs
  resp read:	0.0510 secs, 0.0001 secs, 0.3051 secs

Status code distribution:
  [200]	3000 responses

```

**Redis disabled:**
```
$ hey -n 3000 -c 300 http://127.0.0.1:8080/getthecat/v1/cat/img/rand

Summary:
  Total:	6.6336 secs
  Slowest:	5.3418 secs
  Fastest:	0.0004 secs
  Average:	0.5795 secs
  Requests/sec:	452.2443
  
  Total data:	651731370 bytes
  Size/request:	217243 bytes

Response time histogram:
  0.000 [1]	|
  0.535 [2071]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  1.069 [199]	|■■■■
  1.603 [440]	|■■■■■■■■
  2.137 [101]	|■■
  2.671 [57]	|■
  3.205 [79]	|■■
  3.739 [27]	|■
  4.273 [13]	|
  4.808 [8]	|
  5.342 [4]	|


Latency distribution:
  10% in 0.0049 secs
  25% in 0.0384 secs
  50% in 0.1166 secs
  75% in 0.9937 secs
  90% in 1.5836 secs
  95% in 2.4334 secs
  99% in 3.6398 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0067 secs, 0.0004 secs, 5.3418 secs
  DNS-lookup:	0.0000 secs, 0.0000 secs, 0.0000 secs
  req write:	0.0006 secs, 0.0000 secs, 0.0622 secs
  resp wait:	0.3018 secs, 0.0002 secs, 4.4669 secs
  resp read:	0.2698 secs, 0.0000 secs, 4.6606 secs

Status code distribution:
  [200]	2983 responses
  [404]	17 responses
```

## TODO's:
 - :warning: This project really need more tests and benchmarks
 - :warning: Also need more documentation
 - Image collection engines need to be optimized for better performance
 - May be migration to another database engine?
 - Need to remove some dependencies to reduce size
 
 ## Contributing:
1. Fork it
2. Clone it: `git clone https://github.com/dokzlo13/getthecat`
3. Create your feature branch: `git checkout -b my-new-feature`
4. Make changes and add them: `git add .`
5. Commit: `git commit -m 'Awesome cat api feature'`
6. Push: `git push origin my-new-feature`
7. Pull request
