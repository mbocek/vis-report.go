package dbf

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/LindsayBradford/go-dbf/godbf"
)

type Stravnik struct {
	EvCislo       string
	CenovaSkupina string
	Jmeno         string
}

type StravnikList []Stravnik

type Objednavka struct {
	Datum     time.Time
	DatumACas time.Time
	Druh      string
	EvCislo   string
	Pocet     int
}

type ObjednavkaList []Objednavka

func (r ObjednavkaList) Len() int { return len(r) }
func (r ObjednavkaList) Less(i, j int) bool {
	result := r[i].EvCislo < r[j].EvCislo
	if r[i].EvCislo == r[j].EvCislo {
		if r[i].Datum == r[j].Datum {
			if r[i].Druh == r[j].Druh {
				return r[i].DatumACas.Before(r[j].DatumACas)
			}
			return r[i].Druh < r[j].Druh
		}
		return r[i].Datum.Before(r[j].Datum)
	}
	return result
}
func (r ObjednavkaList) Swap(i, j int) { r[i], r[j] = r[j], r[i] }

type Jidelnicek struct {
	Datum time.Time
	Druh  string
	Nazev string
	Row   Row
}

type JidelnicekList []Jidelnicek
type Row map[string]string

func ReadStravnik(path string) (StravnikList, error) {
	filePath := fmt.Sprintf("%s%s", path, "/stravnik.dbf")
	dbfTable, err := godbf.NewFromFile(filePath, "windows-1250")
	if err != nil {
		return nil, err
	}
	log.Printf("Reading stravnik records: %d", dbfTable.NumberOfRecords())
	stravnikList := make(StravnikList, dbfTable.NumberOfRecords())
	for i := 0; i < dbfTable.NumberOfRecords(); i++ {
		var stravnik Stravnik
		if dbfTable.RowIsDeleted(i) {
			continue
		}

		stravnik.EvCislo, err = dbfTable.FieldValueByName(i, "EV_CISLO")
		if err != nil {
			return nil, err
		}
		stravnik.CenovaSkupina, err = dbfTable.FieldValueByName(i, "CEN_SKUP")
		if err != nil {
			return nil, err
		}
		stravnik.Jmeno, err = dbfTable.FieldValueByName(i, "JMENO")
		if err != nil {
			return nil, err
		}
		stravnikList[i] = stravnik
	}
	return stravnikList, nil
}

func ReadObjednavka(path string, dateFrom, dateTo time.Time) (ObjednavkaList, error) {
	filePath := fmt.Sprintf("%s%s", path, "/objednav.dbf")
	dbfTable, err := godbf.NewFromFile(filePath, "windows-1250")
	if err != nil {
		return nil, err
	}
	log.Printf("Reading objednavka records: %d", dbfTable.NumberOfRecords())
	objednavkaList := make(ObjednavkaList, 0)
	for i := 0; i < dbfTable.NumberOfRecords(); i++ {
		var objednavka Objednavka
		if dbfTable.RowIsDeleted(i) {
			continue
		}

		datum, err := dbfTable.FieldValueByName(i, "DATUM")
		if err != nil {
			return nil, err
		}
		objednavka.Datum, err = time.Parse("20060102", datum)
		if err != nil {
			return nil, err
		}
		if objednavka.Datum.After(dateTo) || objednavka.Datum.Before(dateFrom) {
			continue
		}
		objednavka.EvCislo, err = dbfTable.FieldValueByName(i, "EV_CISLO")
		if err != nil {
			return nil, err
		}
		objednavka.Druh, err = dbfTable.FieldValueByName(i, "DRUH")
		if err != nil {
			return nil, err
		}

		pocet, err := dbfTable.FieldValueByName(i, "POCET")
		if err != nil {
			return nil, err
		}
		objednavka.Pocet = ConvertToInt(pocet)
		if objednavka.Pocet == 0 {
			continue
		}

		datumACas, err := dbfTable.FieldValueByName(i, "DATCAS_OBJ")
		if err != nil {
			return nil, err
		}
		objednavka.DatumACas, err = time.Parse("20060201150405", datumACas)
		if err != nil {
			return nil, err
		}

		objednavkaList = append(objednavkaList, objednavka)
	}
	sort.Sort(ObjednavkaList(objednavkaList))
	return objednavkaList, nil
}

func ReadJidelnicek(path string, dateFrom, dateTo time.Time) (JidelnicekList, error) {
	filePath := fmt.Sprintf("%s%s", path, "/jidelnic.dbf")
	dbfTable, err := godbf.NewFromFile(filePath, "windows-1250")
	if err != nil {
		return nil, err
	}
	log.Printf("Reading jidelnicek records: %d", dbfTable.NumberOfRecords())
	jidelnicekList := make(JidelnicekList, 0)
	for i := 0; i < dbfTable.NumberOfRecords(); i++ {
		var jidelnicek Jidelnicek
		if dbfTable.RowIsDeleted(i) {
			continue
		}

		datum, err := dbfTable.FieldValueByName(i, "DATUM")
		if err != nil {
			return nil, err
		}
		jidelnicek.Datum, err = time.Parse("20060102", datum)
		if err != nil {
			return nil, err
		}
		if jidelnicek.Datum.After(dateTo) || jidelnicek.Datum.Before(dateFrom) {
			continue
		}

		jidelnicek.Row = dbfTable.GetRowAsMap(i)

		jidelnicek.Druh, err = dbfTable.FieldValueByName(i, "DRUH")
		if err != nil {
			return nil, err
		}

		jidelnicek.Nazev, err = dbfTable.FieldValueByName(i, "NAZEV")
		if err != nil {
			return nil, err
		}

		jidelnicekList = append(jidelnicekList, jidelnicek)
	}
	return jidelnicekList, nil
}

func ConvertToFloat64(number string) float64 {
	n, err := strconv.ParseFloat(number, 64)
	if err != nil {
		message, _ := fmt.Printf("Number: %s is not a float", number)
		panic(message)
	}
	return n
}

func ConvertToInt(number string) int {
	n, err := strconv.Atoi(number)
	if err != nil {
		message, _ := fmt.Printf("Number: %s is not a number", number)
		panic(message)
	}
	return n
}
