package ipdb

import (
	"crypto/md5"
	"io"
	"os"
)

const SIZE = 256 * 256 * 256

type IPDB struct {
	data []byte
}

func NewIPDB(path string) (*IPDB, error) {
	db := &IPDB{}
	db.create()
	if err := db.load(path); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *IPDB) create() {
	db.data = make([]byte, (SIZE/4)*5)
}

func (db *IPDB) ipToKey(ip []byte) []uint32 {
	mk := md5.Sum(ip)
	for i := 0; i < 1000; i++ {
		mk = md5.Sum(mk[:])
	}
	var keys []uint32
	for i := 0; i < 5; i++ {
		key := uint32(0)
		key += uint32(mk[i*3+0])
		key += uint32(mk[i*3+1]) * 256
		key += uint32(mk[i*3+2]) * 256 * 256
		keys = append(keys, key)
	}
	return keys
}

func (db *IPDB) getMap(table int, key uint32) byte {
	offset := table*SIZE/4 + int(key/4)
	mask := []byte{0b00000011, 0b00001100, 0b00110000, 0b11000000}[key%4]
	v := db.data[offset]
	v &= mask
	v >>= (key % 4) * 2
	return v
}

func (db *IPDB) Get(ip []byte) int {
	value := -1
	for i, key := range db.ipToKey(ip) {
		v := db.getMap(i, key)
		if value == -1 {
			value = int(v)
		} else if value != int(v) {
			return 0
		}
	}
	return value
}

func (db *IPDB) load(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	db.data, err = io.ReadAll(file)
	return err
}
