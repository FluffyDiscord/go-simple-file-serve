## Build notes

File server is using single IP whitelist, don't ask why. For this reason either remove coresponding code or add your IP when building, eg. `build.sh`

## What is my purpose ?

To serve archived files in a bit smarter way then plain old apache directory listing without using database.

## Usage

`./fileserver -p 9001 -b /home/user/root_serve_folder`


Using the example usage command:

Send request where the URI is your file/folder, eg.
```kotlin
GET https://your.domain.com/randomFolder
```
will scan files in
```
/home/user/root_serve_folder/randomFolder
```
and cache the result locally to a file in `./cache` and return JSON in format
```ts
{
  "entries": [
    {
      "isDir": boolean,
      "name": string, // eg. is file then "yourFile.extension" or if dir "directoryName"
      "size": number // in bytes, will be probably wrong for folders, it depends on your OS
    }
  ]
}
```
requesting a file will make the server serve it, just like any other http server, eg.
```
GET https://your.domain.com/randomFolder/document.pdf
```
will bring up your browser PDF viewer

Projects main purpose is to serve galleries, so it comes with a feature, which converts
`.png` and `.gif` images if you request `.jpg` variant of it, so if you request:
```kotlin
GET https://your.domain.com/randomFolder/my_image.jpg
```
and this image does not exist, but there is file with the same name in `.png` or `.gif`,
server will convert that file to `.jpg` and save it as a copy inside the folder and serve it back.

There is also a special function to generate cover/preview image if it's not already there by requesting
```kotlin
GET https://your.domain.com/randomFolder/cover.jpg
```
It looks for a file named `1.jpg` and makes small cover image 320px wide, also saving in the same folder

## About the cache

Cache was implemented because scanning tens of thousands directories on a 5400RPM external drive 
connected to RPi4 via USB is dreadfully slow (takes minutes). With this simple cache first request
is still painful but subsequent ones take only miliseconds, because the cached file is served directly with no overhead.

Cache persists between runs so it's up to you to clear it yourself
