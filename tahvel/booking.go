package tahvel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

var BOOKINGNAME = os.Getenv("BOOKINGNAME")

func init() {
	if BOOKINGNAME == "" {
		slog.Error("booking name must not be empty", slog.String("environment", "BOOKINGNAME"))
		os.Exit(1)
	}
}

type Booking struct {
	Id        int
	Date      time.Time
	DateStr   string
	TimeStart string
	TimeEnd   string
	Rooms     []Room // Has only: Id, RoomCode
	RoomStr   string
}

func (t *Tahvel) Bookings(ctx context.Context, date time.Time) ([]Booking, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/hois_back/timetableevents?page=0&size=2000&from="+date.Format("2006-01-02T15:04:05.000")+"Z", nil)
	if err != nil {
		return nil, fmt.Errorf("crafting request: %w", err)
	}
	req.AddCookie(&http.Cookie{Name: "SESSION", Value: t.Session})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing request: %w", err)
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return nil, fmt.Errorf("bad status")
	}

	var result struct {
		Content []Booking
		Last    bool
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading request body: %w", err)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding request: %w", err)
	}

	if !result.Last {
		return nil, fmt.Errorf("not last page, over 2000 rooms?")
	}

	for i, booking := range result.Content {
		var roomStr []string

		for _, room := range booking.Rooms {

			var pianoStr string
			switch pianoRooms[room.OnlyCode()] {
			default:
				roomStr = append(roomStr, room.RoomCode)
				continue
			case 1:
				pianoStr = "🎹"
			case 2:
				pianoStr = "2️⃣"
			}

			roomStr = append(roomStr, pianoStr+" "+room.RoomCode)
		}

		result.Content[i].RoomStr = strings.Join(roomStr, ",")

		result.Content[i].DateStr = booking.Date.Format("2006-01-02")
	}

	return result.Content, nil
}

func (t *Tahvel) CreateBooking(ctx context.Context, roomId int, start, stop time.Time) error {
	reqData := struct {
		Rooms []int  `json:"rooms"`
		Start string `json:"startTime"`
		Stop  string `json:"endTime"`
	}{
		Rooms: []int{roomId},
		Start: start.Format("2006-01-02T15:04:05.000") + "Z",
		Stop:  stop.Format("2006-01-02T15:04:05.000") + "Z",
	}
	reqDataJ, err := json.Marshal(&reqData)
	if err != nil {
		return fmt.Errorf("marshalling request json body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/hois_back/timetableevents/timetableTimeOccupied", bytes.NewReader(reqDataJ))
	if err != nil {
		return fmt.Errorf("crafting request: %w", err)
	}
	req.AddCookie(&http.Cookie{Name: "SESSION", Value: t.Session})

	req.AddCookie(&http.Cookie{Name: "XSRF-TOKEN", Value: "teine"})
	req.Header.Add("X-XSRF-TOKEN", "teine")

	req.Header.Add("Content-Type", "application/json;charset=UTF-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return fmt.Errorf("bad status, kas ruum broneeriti vahetult enne ära?")
	}

	//

	date, _ := time.Parse("2006-01-02", start.Format("2006-01-02"))

	type ReqDataRoom struct {
		Id int `json:"id"`
	}

	reqData2 := struct {
		Single   bool          `json:"isSingleEvent"`
		Public   bool          `json:"isPublic"`
		Personal bool          `json:"isPersonal"`
		Date     string        `json:"date"`
		Start    string        `json:"startTime"`
		Stop     string        `json:"endTime"`
		Rooms    []ReqDataRoom `json:"rooms"`
		Name     string        `json:"name"`
	}{
		Single:   true,
		Public:   true,
		Personal: true,

		Date:  date.Format("2006-01-02T15:04:05.000") + "Z",
		Start: start.Format("2006-01-02T15:04:05.000") + "Z",
		Stop:  stop.Format("2006-01-02T15:04:05.000") + "Z",

		Rooms: []ReqDataRoom{{Id: roomId}},

		Name: BOOKINGNAME,
	}
	reqDataJ, err = json.Marshal(&reqData2)
	if err != nil {
		return fmt.Errorf("marshalling request json body: %w", err)
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/hois_back/timetableevents", bytes.NewReader(reqDataJ))
	if err != nil {
		return fmt.Errorf("crafting request: %w", err)
	}
	req.AddCookie(&http.Cookie{Name: "SESSION", Value: t.Session})

	req.AddCookie(&http.Cookie{Name: "XSRF-TOKEN", Value: "teine"})
	req.Header.Add("X-XSRF-TOKEN", "teine")

	req.Header.Add("Content-Type", "application/json;charset=UTF-8")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return fmt.Errorf("bad status")
	}

	return nil
}

func (t *Tahvel) CancelBooking(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, baseURL+"/hois_back/timetableevents/"+id+"?version=0", nil)
	if err != nil {
		return fmt.Errorf("crafting request: %w", err)
	}
	req.AddCookie(&http.Cookie{Name: "SESSION", Value: t.Session})

	req.AddCookie(&http.Cookie{Name: "XSRF-TOKEN", Value: "teine"})
	req.Header.Add("X-XSRF-TOKEN", "teine")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return fmt.Errorf("bad status")
	}

	return nil
}
