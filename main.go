package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/mbocek/vis-report.go/dbf"
	"github.com/xuri/excelize/v2"
)

const (
	summary = "Summary"
)

type Report struct {
	owner   string
	evCislo string
	items   ReportItems
	count   int
	amount  float64
}

type ReportItem struct {
	datum   time.Time
	created time.Time
	druh    string
	jidlo   string
	pocet   int
	cena    float64
	suma    float64
}
type ReportItems []ReportItem
type ByDate []ReportItem
type ReportList []Report

func (r ByDate) Len() int { return len(r) }
func (r ByDate) Less(i, j int) bool {
	byDate := r[i].datum.Before(r[j].datum) // && (r[i].druh < r[j].druh)
	if r[i].datum == r[j].datum {
		if r[i].druh == r[j].druh {
			return r[i].created.Before(r[j].created)
		}
		return r[i].druh < r[j].druh
	}
	return byDate
}
func (r ByDate) Swap(i, j int) { r[i], r[j] = r[j], r[i] }

func main() {
	now := time.Now()
	dateFrom := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	dateTo := time.Date(now.Year(), now.Month(), 1, 23, 59, 59, 0, now.Location())
	dateTo = dateTo.AddDate(0, 1, -1)

	flag.String("dateFrom", dateFrom.Format("02-01-2006"), "Start date for reporting")
	flag.String("dateTo", dateTo.Format("02-01-2006"), "End date for reporting")
	dataDir := flag.String("dataDir", "./data", "Directory for data")
	previousMonth := flag.Bool("previous", false, "Previous month")

	flag.Parse()
	if *previousMonth {
		dateFrom = dateFrom.AddDate(0, -1, 0)
		dateTo = time.Date(now.Year(), now.Month(), 1, 23, 59, 59, 0, now.Location())
		dateTo = dateTo.AddDate(0, 0, -1)
	}

	log.Printf("Start date %s\n", dateFrom.Format("02-01-2006"))
	log.Printf("End date %s\n", dateTo.Format("02-01-2006"))

	jidelnicekList, err := dbf.ReadJidelnicek(*dataDir, dateFrom, dateTo)
	if err != nil {
		log.Print(err)
	}

	objednavkaList, err := dbf.ReadObjednavka(*dataDir, dateFrom, dateTo)
	if err != nil {
		log.Print(err)
	}

	stravnikList, err := dbf.ReadStravnik(*dataDir)
	if err != nil {
		log.Print(err)
	}

	report := makeReportData(jidelnicekList, objednavkaList, stravnikList)
	generateReport(report, dateFrom, dateTo)

}

func makeReportData(jidelnicekList dbf.JidelnicekList, objednavkaList dbf.ObjednavkaList, stravnikList dbf.StravnikList) ReportList {
	var reportList ReportList
	log.Printf("Generating report data")

	for _, stravnik := range stravnikList {
		var report Report
		report.owner = stravnik.Jmeno
		report.evCislo = stravnik.EvCislo
		for objednavkaIndex, objednavka := range objednavkaList {
			if objednavka.EvCislo != stravnik.EvCislo {
				continue
			}
			if (objednavkaIndex < len(objednavkaList)-1) &&
				objednavka.EvCislo == objednavkaList[objednavkaIndex+1].EvCislo &&
				objednavka.Datum == objednavkaList[objednavkaIndex+1].Datum &&
				objednavka.Druh == objednavkaList[objednavkaIndex+1].Druh {
				continue
			}

			var reportItem ReportItem
			reportItem.datum = objednavka.Datum
			reportItem.created = objednavka.DatumACas
			reportItem.pocet = objednavka.Pocet
			reportItem.druh = objednavka.Druh
			for _, jidelnicek := range jidelnicekList {
				if jidelnicek.Datum == objednavka.Datum && jidelnicek.Druh == objednavka.Druh {
					reportItem.jidlo = jidelnicek.Nazev
					reportItem.cena = dbf.ConvertToFloat64(jidelnicek.Row["CENA"+stravnik.CenovaSkupina])
					reportItem.suma = reportItem.cena * float64(reportItem.pocet)
					break
				}
			}

			report.items = append(report.items, reportItem)
			report.count += reportItem.pocet
			report.amount += reportItem.suma
		}
		sort.Sort(ByDate(report.items))
		reportList = append(reportList, report)
	}

	return reportList
}

func generateReport(reportList ReportList, dateFrom, dateTo time.Time) {
	f := excelize.NewFile()
	log.Printf("Generating excel report")
	var totalCount int
	var totalAmmount float64
	var evcSkip []string

	for _, report := range reportList {
		if len(report.items) == 0 {
			evcSkip = append(evcSkip, report.evCislo)
			continue
		}
		// 31 chars
		f.NewSheet(report.evCislo)
		err := f.SetColWidth(report.evCislo, "A", "A", 20)
		if err != nil {
			log.Println(err)
		}
		err = f.SetColWidth(report.evCislo, "C", "C", 70)
		if err != nil {
			log.Println(err)
		}

		f.SetCellValue(report.evCislo, "A1", report.owner)
		f.SetCellValue(report.evCislo, "A2", "Datum")
		f.SetCellValue(report.evCislo, "B2", "Druh")
		f.SetCellValue(report.evCislo, "C2", "Jidlo")
		f.SetCellValue(report.evCislo, "D2", "Pocet")
		f.SetCellValue(report.evCislo, "E2", "Cena")
		f.SetCellValue(report.evCislo, "F2", "Suma")

		i := 2
		for _, reportItem := range report.items {
			i++
			f.SetCellValue(report.evCislo, fmt.Sprintf("A%d", i), reportItem.datum)
			f.SetCellValue(report.evCislo, fmt.Sprintf("B%d", i), reportItem.druh)
			f.SetCellValue(report.evCislo, fmt.Sprintf("C%d", i), reportItem.jidlo)
			f.SetCellValue(report.evCislo, fmt.Sprintf("D%d", i), reportItem.pocet)
			f.SetCellValue(report.evCislo, fmt.Sprintf("E%d", i), reportItem.cena)
			f.SetCellValue(report.evCislo, fmt.Sprintf("F%d", i), reportItem.suma)
		}
		i++
		f.SetCellValue(report.evCislo, fmt.Sprintf("A%d", i), "Sumary:")
		f.SetCellFormula(report.evCislo, fmt.Sprintf("D%d", i), fmt.Sprintf("=SUM(D3:D%d)", i-1))
		f.SetCellFormula(report.evCislo, fmt.Sprintf("F%d", i), fmt.Sprintf("=SUM(F3:F%d)", i-1))

		totalCount += report.count
		totalAmmount += report.amount
	}
	log.Printf("Skipping report for EVC: %v because of missing orders.", evcSkip)

	// remove first
	if f.SheetCount > 1 {
		f.SetSheetName("Sheet1", summary)
		err := f.SetColWidth(summary, "A", "A", 50)
		if err != nil {
			log.Println(err)
		}

		i := 0
		for _, report := range reportList {
			if report.amount == 0 {
				continue
			}
			i++
			f.SetCellValue(summary, fmt.Sprintf("A%d", i), report.owner)
			f.SetCellValue(summary, fmt.Sprintf("B%d", i), report.amount)
		}

		i++
		f.SetCellValue(summary, fmt.Sprintf("A%d", i), "Total count:")
		f.SetCellValue(summary, fmt.Sprintf("B%d", i), totalCount)
		i++
		f.SetCellValue(summary, fmt.Sprintf("A%d", i), "Total amount:")
		f.SetCellValue(summary, fmt.Sprintf("B%d", i), totalAmmount)
	}

	// Save spreadsheet by the given path.
	fileName := fmt.Sprintf("report_%s_%s.xlsx", dateFrom.Format("02-01-2006"), dateTo.Format("02-01-2006"))
	if err := f.SaveAs(fileName); err != nil {
		log.Println(err)
	}
}
