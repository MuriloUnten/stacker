package main

import (
)

/**
 * TODO
 * - [ ] implement getter for all versions of an assembly
 * - [ ] make sure recursive bom's are impossible and update tests for that
 * - [ ] implement endpoints for /items/{id}/versions
 * - [x] use decimal types instead of decimal.Decimal's (https://github.com/shopspring/decimal)
 * - [x] Position in bom_line
 * - [ ] think about refactoring images to use a url (that would allow for later refactor to cdn)
 * - [ ] Treat possible active version on item related functions
 * - [x] Implement Store
 * - [ ] Prepare all statements beforehand
 * - [ ] Read about prepared statements in transactions
 * - [x] Enable write ahead logging (WAL)
 * - [x] Think about assembly versioning and implement it
 * - [x] Implement recursive read of BoM tree
 * - [x] Implement migration system
 * - [ ] Implement connection pool
 * - [ ] Add better errors
 * - [ ] think about having timestamps in more things
 * - [x] Write testing framework
 * - [/] Add Tests
 * - [x] Consider using sqlc
 * - [ ] Study and think about context and timeouts
 * - [ ] Add more fields to image table (mime_type, width, height, etc)
 * - [ ] Add optional image input to item creation functions
 * - [ ] Think about how to implement a tracker scanner goroutine
 * - [ ] Implement manufacture orders
 * - [x] Think about manufacture orders having another state maybe called refurbished or something that restores inventory
 * - [x] Rewatch the McMaster-Carr video (https://www.youtube.com/watch?v=-Ln-8QM8KhQ)
 */
func main() {
	store := StoreInit("db/database.db")
	defer store.Deinit()

	server := NewServer(":1337", store)
	server.Run()
}
