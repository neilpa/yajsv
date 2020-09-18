// +build ignore

// generates clones the utf-8 tests data to the other
// unicode encodings and adds BOM variants of each.
package main

import (
    "io/ioutil"
    "log"
    "os"
    "path/filepath"

    "golang.org/x/text/encoding"
    "golang.org/x/text/encoding/unicode"
)


func main() {
    var xforms = []struct {
        dir, bom string
        enc encoding.Encoding
    } {
        { "testdata/utf-16be", "\xFE\xFF", unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM) },
        { "testdata/utf-16le", "\xFF\xFE", unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM) },
    }

    paths, _ := filepath.Glob("testdata/utf-8/*")
    for _, p := range paths {
        src, err := ioutil.ReadFile(p)
        if err != nil {
            log.Fatal(err)
        }

        write("testdata/utf-8_bom", p, "\xEF\xBB\xBF", src)
        for _, xform := range xforms {
            dst, err := xform.enc.NewEncoder().Bytes(src)
            if err != nil {
                log.Fatal(err)
            }
            write(xform.dir, p, "", dst)
            write(xform.dir + "_bom", p, xform.bom, dst)
        }
    }
}

func write(dir, orig, bom string, buf []byte) {
    f, err := os.Create(filepath.Join(dir, filepath.Base(orig)))
    if err != nil {
        log.Fatal(err)
    }
    if _, err = f.Write([]byte(bom)); err != nil {
        log.Fatal(err)
    }
    if _, err = f.Write(buf); err != nil {
        log.Fatal(err)
    }
}
