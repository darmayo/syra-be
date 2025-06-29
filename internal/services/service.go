package services

import (
    "database/sql"
    "log"
    "time"
)

type Service struct {
    DB *sql.DB
}

// NewService initializes a new Service with a database connection.
func NewService(db *sql.DB) *Service {
    return &Service{DB: db}
}

func (s *Service) PerformAction() error {
    // Implement the business logic here
    return nil
}

// FetchAlerts returns alerts only for domains owned by the given user, matching agent_name to domain name.
func (s *Service) FetchAlerts(userID int) ([]map[string]interface{}, error) {
    query := `
        SELECT a.id, a.severity, a.pretext, a.title, a."text", a.rule_id, a."timestamp", a.agent_id, a.agent_name, a.agent_ip, a.manager_name, a.full_log, a.decoder_name, a.protocol, a.srcip, a.url, a.status_code, a."location", a.raw_data
        FROM public.alerts a
        JOIN domain d ON a.agent_name = d.name
        WHERE d.user_id = $1
        ORDER BY a."timestamp" DESC;
    `

    rows, err := s.DB.Query(query, userID)
    if err != nil {
        log.Printf("Error executing query: %v", err)
        return nil, err
    }
    defer rows.Close()

    var alerts []map[string]interface{}
    for rows.Next() {
        var alert = make(map[string]interface{})
        var (
            ruleID, agentID, statusCode sql.NullInt64
            id, severity, pretext, title, text, agentName, agentIP, managerName, fullLog, decoderName, protocol, srcip, url, location, rawData sql.NullString
            timestamp sql.NullTime
        )

        err := rows.Scan(&id, &severity, &pretext, &title, &text, &ruleID, &timestamp, &agentID, &agentName, &agentIP, &managerName, &fullLog, &decoderName, &protocol, &srcip, &url, &statusCode, &location, &rawData)
        if err != nil {
            log.Printf("Error scanning row: %v", err)
            return nil, err
        }

        alert["id"] = id.String
        alert["severity"] = severity.String
        alert["pretext"] = pretext.String
        alert["title"] = title.String
        alert["text"] = text.String
        alert["rule_id"] = ruleID.Int64
        alert["timestamp"] = timestamp.Time
        alert["agent_id"] = agentID.Int64
        alert["agent_name"] = agentName.String
        alert["agent_ip"] = agentIP.String
        alert["manager_name"] = managerName.String
        alert["full_log"] = fullLog.String
        alert["decoder_name"] = decoderName.String
        alert["protocol"] = protocol.String
        alert["srcip"] = srcip.String
        alert["url"] = url.String
        alert["status_code"] = statusCode.Int64
        alert["location"] = location.String
        alert["raw_data"] = rawData.String

        alerts = append(alerts, alert)
    }

    if err := rows.Err(); err != nil {
        log.Printf("Error iterating rows: %v", err)
        return nil, err
    }

    return alerts, nil
}

// AddDomain inserts a new domain for the given user.
func (s *Service) AddDomain(name, url string, userID int) (int, string, string, error) {
    query := `
        INSERT INTO domain (name, url, created_at, user_id)
        VALUES ($1, $2, $3, $4)
        RETURNING id, created_at;
    `
    var id int
    var createdAt string
    err := s.DB.QueryRow(query, name, url, time.Now(), userID).Scan(&id, &createdAt)
    if err != nil {
        log.Printf("Error inserting domain: %v", err)
        return 0, "", "", err
    }
    return id, name, url, nil
}

// GetDomains fetches all domains for the given user.
func (s *Service) GetDomains(userID int) ([]map[string]interface{}, error) {
    query := `
        SELECT id, name, url, created_at
        FROM domain
        WHERE user_id = $1;
    `
    rows, err := s.DB.Query(query, userID)
    if err != nil {
        log.Printf("Error fetching domains: %v", err)
        return nil, err
    }
    defer rows.Close()

    var domains []map[string]interface{}
    for rows.Next() {
        var (
            id        int
            name      string
            url       string
            createdAt time.Time
        )
        err := rows.Scan(&id, &name, &url, &createdAt)
        if err != nil {
            log.Printf("Error scanning row: %v", err)
            return nil, err
        }
        domain := map[string]interface{}{
            "id":        id,
            "name":      name,
            "url":       url,
            "createdAt": createdAt.Format(time.RFC3339),
        }
        domains = append(domains, domain)
    }
    if err := rows.Err(); err != nil {
        log.Printf("Error iterating rows: %v", err)
        return nil, err
    }
    return domains, nil
}

// DeleteDomain deletes a domain by id only if it belongs to the user.
func (s *Service) DeleteDomain(domainId int, userID int) error {
    query := `
        DELETE FROM domain
        WHERE id = $1 AND user_id = $2;
    `
    result, err := s.DB.Exec(query, domainId, userID)
    if err != nil {
        log.Printf("Error deleting domain: %v", err)
        return err
    }
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        log.Printf("Error checking rows affected: %v", err)
        return err
    }
    if rowsAffected == 0 {
        log.Printf("No domain found with id: %d for user: %d", domainId, userID)
        return sql.ErrNoRows
    }
    return nil
}

// UpsertUser inserts or updates a user by email.
func (s *Service) UpsertUser(name, email string) error {
    query := `
        INSERT INTO "user" (name, email)
        VALUES ($1, $2)
        ON CONFLICT (email) DO UPDATE
        SET name = EXCLUDED.name;
    `
    _, err := s.DB.Exec(query, name, email)
    return err
}

// GetUserByEmail fetches a user by email.
func (s *Service) GetUserByEmail(email string) (map[string]interface{}, error) {
    query := `SELECT id, name, email, created_by FROM "user" WHERE email = $1`
    row := s.DB.QueryRow(query, email)
    var id int
    var name, dbEmail, createdBy string
    err := row.Scan(&id, &name, &dbEmail, &createdBy)
    if err != nil {
        return nil, err
    }
    return map[string]interface{}{
        "id":    id,
        "name":  name,
        "email": dbEmail,
        "createdBy": createdBy,
    }, nil
}