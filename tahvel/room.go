package tahvel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type (
	Room struct {
		Id       int
		RoomCode string // number, often with type
		RoomName string // type
		// BuildingCode string // dupe
		BuildingName  string
		Times         []string
		Places        int  // aka seats
		IsUsedInStudy bool // true = available
		Equipment     []EquipmentListing

		// added for internal data passing
		ConflictReason string
		MissingACL     bool
		PianoCount     int
		// added at render, do not use elsewhere
		ResolvedEquipmnet string
	}
	EquipmentListing struct {
		Equipment      string
		EquipmentCount int
	}
)

func (t *Tahvel) GetRooms(ctx context.Context, date time.Time) ([]Room, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/hois_back/timetableevents/rooms", nil)
	if err != nil {
		return nil, fmt.Errorf("crafting request: %w", err)
	}
	req.AddCookie(&http.Cookie{Name: "SESSION", Value: t.Session})

	dateS := date.Format("2006-01-02T15:04:05.000") + "Z"

	query := req.URL.Query()
	for k, v := range map[string]string{
		"isBusyRoom":       "false",
		"isFreeRoom":       "true",
		"isPartlyBusyRoom": "true",
		"page":             "0",
		"size":             "2000", // max
		"thru":             dateS,
		"from":             dateS,
	} {
		query.Add(k, v)
	}
	req.URL.RawQuery = query.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing request: %w", err)
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return nil, fmt.Errorf("bad status")
	}

	var roomResult struct {
		Content []Room
		Last    bool
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading request body: %w", err)
	}
	if err := json.Unmarshal(body, &roomResult); err != nil {
		return nil, fmt.Errorf("decoding request: %w", err)
	}

	if !roomResult.Last {
		return nil, fmt.Errorf("not last page, over 2000 rooms?")
	}

	for i, room := range roomResult.Content {
		if count, ok := pianoRooms[room.OnlyCode()]; ok {
			roomResult.Content[i].PianoCount = count
		}
	}

	return roomResult.Content, nil
}

func (r *Room) OnlyCode() string {
	c, _, _ := strings.Cut(r.RoomCode, " ")
	return c
}

func GetEquipment(ctx context.Context) (map[string]string, error) {
	equipment := make(map[string]string)

	for _, class := range []string{"SEADMED"} {
		resp, err := http.Get(baseURL + "/hois_back/autocomplete/classifiers?mainClassCode=" + class)
		if err != nil {
			return nil, fmt.Errorf("getting equipment: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading equipment list: %w", err)
		}

		type (
			EquipmentListItem struct {
				Code   string
				NameEt string
			}
			EquipmentList []EquipmentListItem
		)

		var el EquipmentList
		if err := json.Unmarshal(body, &el); err != nil {
			return nil, fmt.Errorf("unmarshaling equipment list: %w", err)
		}

		for _, e := range el {
			equipment[e.Code] = e.NameEt
		}
	}

	return equipment, nil
}
