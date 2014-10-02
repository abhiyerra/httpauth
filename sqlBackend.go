package httpauth

import (
    "database/sql"
)

// SqlAuthBackend database and database connection information.
type SqlAuthBackend struct {
    driverName     string
    dataSourceName string
    db             *sql.DB

    // prepared statements
    userStmt       *sql.Stmt
    usersStmt      *sql.Stmt
    insertStmt     *sql.Stmt
    updateStmt     *sql.Stmt
    deleteStmt     *sql.Stmt
}

// NewSqlAuthBackend initializes a new backend by testing the database
// connection and making sure the storage table exists. The table is called
// goauth.
//
// Returns an error if connecting to the database fails, pinging the database
// fails, or creating the table fails.
//
// This uses the databases/sql package to open a connection. Its parameters
// should match the sql.Open function. See
// http://golang.org/pkg/database/sql/#Open for more information.
//
// Be sure to import "database/sql" and your driver of choice. If you're not
// using sql for your own purposes, you'll need to use the underscore to import
// for side effects; see http://golang.org/doc/effective_go.html#blank_import.
func NewSqlAuthBackend(driverName, dataSourceName string) (b SqlAuthBackend, e error) {
    b.driverName = driverName
    b.dataSourceName = dataSourceName
    db, err := sql.Open(driverName, dataSourceName)
    if err != nil {
        return b, err
    }
    err = db.Ping()
    if err != nil {
        return b, err
    }
    b.db = db
    _, err = db.Exec(`create table if not exists goauth (Username varchar(255), Email varchar(255), Hash varchar(255), Role varchar(255), primary key (Username))`)
    if err != nil {
        return b, err
    }

    // prepare statements
    b.userStmt, err = db.Prepare(`select Email, Hash, Role from goauth where Username=?`)
    if err != nil {
        return b, err
    }
    b.usersStmt, err = db.Prepare(`select Username, Email, Hash, Role from goauth`)
    if err != nil {
        return b, err
    }
    b.insertStmt, err = db.Prepare(`insert into goauth (Username, Email, Hash, Role) values (?, ?, ?, ?)`)
    if err != nil {
        return b, err
    }
    b.updateStmt, err = db.Prepare(`update goauth set Email=?, Hash=?, Role=? where Username=?`)
    if err != nil {
        return b, err
    }
    b.deleteStmt, err = db.Prepare(`delete from goauth where Username=?`)
    if err != nil {
        return b, err
    }

    return b, nil
}

// User returns the user with the given username. Error is set to
// ErrMissingUser if user is not found.
func (b SqlAuthBackend) User(username string) (user UserData, e error) {
    row := b.userStmt.QueryRow(username)
    err := row.Scan(&user.Email, &user.Hash, &user.Role)
    if err != nil {
        if err == sql.ErrNoRows {
            return user, ErrMissingUser
        }
        return user, err
    }
    user.Username = username
    return user, nil
}

// Users returns a slice of all users.
func (b SqlAuthBackend) Users() (us []UserData, e error) {
    rows, err := b.usersStmt.Query()
    if err != nil {
        return us, err
    }
    var (
        username, email, role string
        hash                  []byte
    )
    for rows.Next() {
        err = rows.Scan(&username, &email, &hash, &role)
        if err != nil {
            return us, err
        }
        us = append(us, UserData{username, email, hash, role})
    }
    return us, nil
}

// SaveUser adds a new user, replacing one with the same username.
func (b SqlAuthBackend) SaveUser(user UserData) (err error) {
    if _, err := b.User(user.Username); err == nil {
        _, err = b.updateStmt.Exec(user.Email, user.Hash, user.Role, user.Username)
    } else {
        _, err = b.insertStmt.Exec(user.Username, user.Email, user.Hash, user.Role)
    }
    return
}

// DeleteUser removes a user, raising ErrDeleteNull if that user was missing.
func (b SqlAuthBackend) DeleteUser(username string) error {
    result, err := b.deleteStmt.Exec(username)
    if err != nil {
        return err
    }
    rows, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rows == 0 {
        return ErrDeleteNull
    }
    return err
}

// Close cleans up the backend by terminating the database connection.
func (b SqlAuthBackend) Close() {
    b.db.Close()
}
