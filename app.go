package main

import (
	"crypto/md5"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/django/v3"
	"github.com/google/uuid"
	"io"
	"log"
	"mdict-http/services/dict"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	Dicts = map[string]dict.Dict{}

	//go:embed views
	ViewFs embed.FS
)

func main() {
	RegisterDicts()
	engine := django.NewPathForwardingFileSystem(http.FS(ViewFs), "/views", ".html")
	engine.Reload(true)

	app := fiber.New(fiber.Config{
		Views: engine,
	})
	app.Get("/", func(c fiber.Ctx) error {
		return c.Render("home.index", fiber.Map{})
	})
	app.Get("/api/dicts", func(c fiber.Ctx) error {
		var ds []dict.Dict
		for _, d := range Dicts {
			ds = append(ds, d)
		}
		return c.JSON(ds)
	})
	app.Get("/resources/dicts/:dictid", func(c fiber.Ctx) error {
		var d dict.Dict
		id := c.Params("dictid")

		if _, ok := Dicts[c.Params("dictid")]; ok {
			d = Dicts[id]
		} else {
			return c.SendStatus(404)
		}
		p := strings.Replace(c.Query("path"), fmt.Sprintf("/resources/dicts/%s", id), "", 1)

		_, err := os.Stat(filepath.Join(d.Path, p))
		if err == nil {
			return c.SendFile(filepath.Join(d.Path, p))
		}

		var result []byte
		for _, mdd := range d.Mdds {
			result, err = mdd.Lookup(c.Query("path"))

			if err != nil {
				//re search new path
				result, err = mdd.Lookup("\\" + c.Query("path"))
				if err != nil {
					continue
				} else {
					err = nil
					break
				}
			} else {
				err = nil
				break
			}
		}

		if result == nil || err != nil {
			if err != nil {
				log.Printf("mdict.LookupResource failed, key [%s] , error: %s", c.Query("path"), err.Error())
			}
			return c.SendStatus(404)
		}

		return c.Send(result)

	})
	app.Get("/api/query", func(c fiber.Ctx) error {
		fmt.Printf("query:%s\n", c.Query("keyword"))
		keyword := c.Query("keyword")
		result := make([]map[string]interface{}, 0)
		for _, id := range strings.Split(c.Query("dict_ids"), ",") {
			if _, ok := Dicts[id]; ok {
				d := Dicts[id]
				desc, err := d.Dictionary.Lookup(keyword)
				if err != nil {
					result = append(result, map[string]interface{}{
						"id":     d.ID,
						"error":  err.Error(),
						"result": "",
					})
				}
				descStr := string(desc)
				if strings.HasPrefix(descStr, "@@@LINK=") {
					newKw := strings.TrimPrefix(descStr, "@@@LINK=")
					newKw = strings.TrimRight(newKw, "\r\n\000")
					desc, _ = d.Dictionary.Lookup(newKw)
					descStr = string(desc)
				}
				descStr = d.AfterRecordFound(&d, descStr)

				result = append(result, map[string]interface{}{
					"id":     d.ID,
					"result": descStr,
					"error":  nil,
				})

			}

		}
		return c.JSON(result)
	})
	log.Fatal(app.Listen(":3233"))
}
func RegisterDicts() {
	es := os.Getenv("MDICT_PATH")
	if len(es) == 0 {
		wd, _ := os.Getwd()
		es = filepath.Join(wd, "dicts")
	}

	files, err := os.ReadDir(es)
	if err != nil {
		log.Fatalf("read dict error,path:%s,err:%v", es, err)
	}

	for _, file := range files {
		if !file.IsDir() {
			log.Printf("dicts should be in sub folder , [%s] ignored", file.Name())
			continue
		}
		fs, err := os.ReadDir(filepath.Join(es, file.Name()))
		if err != nil {
			log.Fatalf("read dict error,path:%s,err:%v", filepath.Join(es, file.Name()), err)
		}
		var dictFiles []string

		meta := make(map[string]interface{})

		var mdict *dict.Mdict
		mdds := make([]*dict.Mdict, 0)
		for _, f := range fs {

			dictFiles = append(dictFiles, f.Name())
			if filepath.Ext(f.Name()) == ".mdx" || filepath.Ext(f.Name()) == ".mdd" {
				mdictT, err := dict.New(filepath.Join(es, file.Name(), f.Name()))
				if err != nil {
					log.Fatalf("new dict error,path:%s,err:%v", filepath.Join(es, file.Name(), f.Name()), err)
				}
				err = mdictT.BuildIndex()
				if err != nil {
					log.Fatalf("build index error,path:%s,err:%v", filepath.Join(es, file.Name(), f.Name()), err)
				}
				if mdictT.IsMDD() {
					mdds = append(mdds, mdictT)
				} else {
					mdict = mdictT
					meta["title"] = mdict.Title()
					meta["version"] = mdict.Version()
					meta["description"] = mdict.Description()
					meta["date"] = mdict.CreationDate()
					meta["num"] = mdict.EntriesNum()
					if _, ok := meta["name"]; !ok {
						meta["name"] = mdict.Name()
					}
					meta["is_mdd"] = mdict.IsMDD()
					meta["is_encrypted"] = mdict.IsRecordEncrypted()

					hash := md5.New()
					tf, _ := os.Open(filepath.Join(es, file.Name(), f.Name()))
					io.Copy(hash, tf)
					meta["id"] = hex.EncodeToString(hash.Sum(nil))
				}

			}

			if f.Name() == "info.json" {
				bs, err := os.ReadFile(filepath.Join(es, file.Name(), f.Name()))
				if err != nil {
					log.Fatalf("read info error,path:%s,err:%v", filepath.Join(es, file.Name(), f.Name()), err)
				}
				var t map[string]interface{}
				err = json.Unmarshal(bs, &t)
				if err != nil {
					log.Fatalf("unmarshal info error,path:%s,err:%v", filepath.Join(es, file.Name(), f.Name()), err)
				}
				for k, v := range t {
					meta[k] = v
				}
			}

		}

		id := uuid.New().String()
		if meta["id"] != nil {
			id = meta["id"].(string)
		}
		Dicts[id] = dict.Dict{
			ID:         id,
			Name:       meta["name"].(string),
			Meta:       meta,
			Dictionary: mdict,
			Files:      dictFiles,
			Mdds:       mdds,
			Num:        meta["num"].(int64),
			Path:       filepath.Join(es, file.Name()),
			AfterRecordFound: func(dict *dict.Dict, content string) string {

				fontReg := regexp.MustCompile(`url\(["|'](\S+\.(ttf|otf|woff|woff2))["|']\)`)
				cssReg := regexp.MustCompile(`href=["|'](\S+\.css)["|']`)
				entryReg := regexp.MustCompile(`href="entry://([\w#_ -]+)"`)
				imageReg := regexp.MustCompile(`src=["|'](\S+\.(png|jpg|gif|jpeg|svg|bmp))["|']`)
				jsReg := regexp.MustCompile(`src="(\S+\.js)"`)
				audioReg := regexp.MustCompile(`href=["|'](sound://\S+\.(mp3|aac))["|']`)

				content = fontReg.ReplaceAllString(content, `url("/resources/dicts/`+dict.ID+`?path=$1")`)
				content = cssReg.ReplaceAllString(content, `href="/resources/dicts/`+dict.ID+`?path=$1"`)
				content = entryReg.ReplaceAllString(content, `href="/word?id=$1&dict_id=`+dict.ID+`" data-entry="`+dict.ID+`-$1"`)
				content = imageReg.ReplaceAllString(content, `src="/resources/dicts/`+dict.ID+`?path=$1"`)
				content = jsReg.ReplaceAllString(content, `src="/resources/dicts/`+dict.ID+`?path=$1"`)
				content = audioReg.ReplaceAllString(content, `href="/resources/dicts/`+dict.ID+`?path=$1"`)

				return content

			},
		}
	}
}
