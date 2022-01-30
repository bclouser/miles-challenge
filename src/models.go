package main

import (
	"encoding/json"
	"fmt"
	"time"
)

type StravaAPIClient struct {
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	TokenEndpoint string `json:"token_endpoint"`
}

// UnixTime lets us use json marshalling in golang's json marshal/unmarshal
type UnixTime struct {
	time.Time
}

func (u *UnixTime) UnmarshalJSON(b []byte) error {
	var timestamp int64
	err := json.Unmarshal(b, &timestamp)
	if err != nil {
		return err
	}
	u.Time = time.Unix(timestamp, 0)
	return nil
}

// MarshalJSON turns our time.Time back into an int
func (u UnixTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", (u.Time.Unix()))), nil
}

type StravaAthlete struct {
	ID            int       `json:"id"`
	Username      string    `json:"username"`
	ResourceState int       `json:"resource_state"`
	Firstname     string    `json:"firstname"`
	Lastname      string    `json:"lastname"`
	Bio           string    `json:"bio"`
	City          string    `json:"city"`
	State         string    `json:"state"`
	Country       string    `json:"country"`
	Sex           string    `json:"sex"`
	Premium       bool      `json:"premium"`
	Summit        bool      `json:"summit"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	BadgeTypeID   int       `json:"badge_type_id"`
	Weight        float32   `json:"weight"`
	ProfileMedium string    `json:"profile_medium"`
	Profile       string    `json:"profile"`
	Friend        string    `json:"friend"`
	Follower      string    `json:"follower"`
}

type StravaUser struct {
	TokenType    string        `json:"token_type"`
	ExpiresAt    UnixTime      `json:"expires_at"`
	ExpiresIn    UnixTime      `json:"expires_in"`
	RefreshToken string        `json:"refresh_token"`
	AccessToken  string        `json:"access_token"`
	Athlete      StravaAthlete `json:"athlete"`
}

type MetaAthlete struct {
	ID            int64 `json:"id"`
	ResourceState int   `json:"resource_state"`
}

type MetaActivity struct {
	ResourceState int         `json:"resource_state"`
	Athlete       MetaAthlete `json:"athlete"`
}

type MapPreview struct {
	ID              string `json:"id"`
	SummaryPolyline string `json:"summary_polyline"`
	ResourceState   int    `json:"resource_state"`
}

type SummaryActivity struct {
	MetaActivity
	ID                         int64      `json:"id"`
	Name                       string     `json:"name"`
	Distance                   float32    `json:"distance"`
	MovingTime                 int        `json:"moving_time"`
	ElapsedTime                int        `json:"elapsed_time"`
	TotalElevationGain         float32    `json:"total_elevation_gain"`
	Type                       string     `json:"type"`
	WorkoutType                int        `json:"workout_type"`
	StartDate                  time.Time  `json:"start_date"`
	StartDateLocal             time.Time  `json:"start_date_local"`
	Timezone                   string     `json:"timezone"`
	UTCOffset                  float32    `json:"utc_offset"`
	LocationCity               string     `json:"location_city"`
	LocationState              string     `json:"location_state"`
	LocationCountry            string     `json:"location_country"`
	AchievementCount           int        `json:"achievement_count"`
	KudosCount                 int        `json:"kudos_count"`
	CommentCount               int        `json:"comment_count"`
	AthleteCount               int        `json:"athlete_count"`
	PhotoCount                 int        `json:"photo_count"`
	Map                        MapPreview `json:"map"`
	Trainer                    bool       `json:"trainer"`
	Commute                    bool       `json:"commute"`
	Manual                     bool       `json:"manual"`
	Private                    bool       `json:"private"`
	Visibility                 string     `json:"visibility"`
	Flagged                    bool       `json:"flagged"`
	GearID                     string     `json:"gear_id"`
	StartLatlng                []float32  `json:"start_latlng"`
	EndLatLng                  []float32  `json:"end_latlng"`
	StartLatitude              float32    `json:"start_latitude"`
	StartLongitude             float32    `json:"start_longitude"`
	AverageSpeed               float32    `json:"average_speed"`
	MaxSpeed                   float32    `json:"max_speed"`
	AverageCadence             float32    `json:"average_cadence"`
	HasHeartrate               bool       `json:"has_heartrate"`
	HeartrateOptOut            bool       `json:"heartrate_opt_out"`
	DisplayHideHeartrateOption bool       `json:"display_hide_heartrate_option"`
	ElevHigh                   float32    `json:"elev_high"`
	ElevLow                    float32    `json:"elev_low"`
	UploadID                   int64      `json:"upload_id"`
	UploadIDStr                string     `json:"upload_id_str"`
	ExternalID                 string     `json:"external_id"`
	FromAcceptedTag            bool       `json:"from_accepted_tag"`
	PRCount                    int        `json:"pr_count"`
	TotalPhotoCount            int        `json:"total_photo_count"`
	HasKudoed                  bool       `json:"has_kudoed"`
}
