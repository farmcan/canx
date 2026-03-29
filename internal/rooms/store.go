package rooms

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Room struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	RunID     string    `json:"run_id,omitempty"`
	RepoRoot  string    `json:"repo_root,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type Message struct {
	ID            string    `json:"id"`
	RoomID        string    `json:"room_id"`
	ParticipantID string    `json:"participant_id"`
	Role          string    `json:"role"`
	Kind          string    `json:"kind"`
	Body          string    `json:"body"`
	TaskID        string    `json:"task_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type Store struct {
	Root string
}

func NewStore(root string) Store {
	return Store{Root: root}
}

func (s Store) SaveRoom(room Room) error {
	now := time.Now()
	if room.CreatedAt.IsZero() {
		room.CreatedAt = now
	}
	room.UpdatedAt = now
	if room.ID == "" {
		room.ID = "room-" + randomID()
	}
	dir := filepath.Join(s.Root, ".canx", "rooms", room.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(room, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "room.json"), data, 0o644)
}

func (s Store) ListRooms() ([]Room, error) {
	root := filepath.Join(s.Root, ".canx", "rooms")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var rooms []Room
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		room, err := s.LoadRoom(entry.Name())
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].UpdatedAt.After(rooms[j].UpdatedAt)
	})
	return rooms, nil
}

func (s Store) LoadRoom(roomID string) (Room, error) {
	data, err := os.ReadFile(filepath.Join(s.Root, ".canx", "rooms", roomID, "room.json"))
	if err != nil {
		return Room{}, err
	}
	var room Room
	if err := json.Unmarshal(data, &room); err != nil {
		return Room{}, err
	}
	return room, nil
}

func (s Store) AppendMessage(roomID string, message Message) (Message, error) {
	if message.ID == "" {
		message.ID = "msg-" + randomID()
	}
	message.RoomID = roomID
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}
	dir := filepath.Join(s.Root, ".canx", "rooms", roomID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Message{}, err
	}
	file, err := os.OpenFile(filepath.Join(dir, "messages.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return Message{}, err
	}
	defer file.Close()
	data, err := json.Marshal(message)
	if err != nil {
		return Message{}, err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return Message{}, err
	}
	return message, nil
}

func (s Store) ListMessages(roomID string) ([]Message, error) {
	file, err := os.Open(filepath.Join(s.Root, ".canx", "rooms", roomID, "messages.jsonl"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()
	var messages []Message
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var message Message
		if err := json.Unmarshal(scanner.Bytes(), &message); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return messages, nil
}

func randomID() string {
	var raw [6]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return time.Now().Format("150405000000")
	}
	return hex.EncodeToString(raw[:])
}
