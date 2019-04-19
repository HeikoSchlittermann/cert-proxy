package shared

import "os"

func Mkdir(dir string) error {

	err := os.Mkdir(dir, 0777)

	if err != nil && os.IsExist(err) {
		stat, err := os.Stat(dir)
		if err != nil {
			return err
		}
		if stat.IsDir() {
			return nil
		}
	}
	return err
}
