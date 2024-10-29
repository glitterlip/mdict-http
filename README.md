# Mdict-Http

export mdict dictionaries as a http service
> based on [https://github.com/terasum/medict](https://github.com/terasum/medict)

## Usage

download the binary `mdict-linux` for linux or `mdict-mac` for mac(apple silicon)

`/path/to/mdx/folder` should point to a folder like dicts folder in this project or in the same folder with the binary

for linux
```shell
chmod +x ./mdict-linux && export MDICT_PATH=/path/to/mdx/folder && export MDICT_PORT=3223 && ./mdict-linux

```

for mac
```shell
chmod +x ./mdict-mac && export MDICT_PATH=/path/to/mdx/folder && export MDICT_PORT=3223 && ./mdict-mac
```

## API

`/api/dicts` list all dictionaries

`/api/query?dict_ids=59adf21359b83709777530abde6cc7a3&keyword=%E3%82%82%E3%82%89%E3%81%84` search keyword with dicts

`/api/dict/:dict/:word` get the word detail