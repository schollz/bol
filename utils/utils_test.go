package utils

import "testing"

func TestParseDate(t *testing.T) {
	correctDate := "2005-04-07 22:13:13 +0200 +0200"
	date, _ := ParseDate("Thu, 07 Apr 2005 22:13:13 +0200")
	if date.String() != correctDate {
		t.Errorf("Expected %s and got %s", correctDate, date.String())
	}

	correctDate = "2005-04-07 22:13:13 +0000 UTC"
	date, _ = ParseDate("2005-04-07 22:13:13")
	if date.String() != correctDate {
		t.Errorf("Expected %s and got %s", correctDate, date.String())
	}

	correctDate = "2005-04-07 22:13:00 +0000 UTC"
	date, _ = ParseDate("2005-04-07 22:13")
	if date.String() != correctDate {
		t.Errorf("Expected %s and got %s", correctDate, date.String())
	}

}

func TestFormatDate(t *testing.T) {
	testDate := "Thu Apr 07 22:13:13 2005 +0200"
	date, err := ParseDate(testDate)
	if err != nil {
		t.Errorf(err.Error())
	}
	if FormatDate(date) != "2005-04-07 22:13:13" {
		t.Errorf("Expected %s and got %s", testDate, FormatDate(date))
	}

}
