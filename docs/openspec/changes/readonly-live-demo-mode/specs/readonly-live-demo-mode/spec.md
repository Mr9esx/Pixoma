## ADDED Requirements

### Requirement: Live demo startup flag
Pixoma SHALL support a `-livedemo` startup flag that activates the isolated readonly live demo runtime without creating, opening, or persisting a normal bootstrap database, business database, or Blob store. An in-process ephemeral datastore MAY be used for demo state.

#### Scenario: Start with live demo flag
- **WHEN** the operator starts Pixoma with `-livedemo`
- **THEN** the HTTP service starts in live demo mode
- **AND** the process does not create or read a persistent bootstrap database, business database, or Blob store

### Requirement: Fixed demo account
Live demo mode SHALL authenticate the fixed account `admin` with password `123456` and SHALL prominently display the credentials reminder on the login page.

#### Scenario: Demo administrator signs in
- **WHEN** the visitor submits `admin` and `123456`
- **THEN** Pixoma grants a demo administrator session
- **AND** the login page has displayed a visible reminder containing both credentials

### Requirement: In-memory demo dataset
Live demo mode SHALL seed representative demo data in process memory, including edges, cases, tasks, users, channels, topics, and statistics, and SHALL NOT connect demo data to real infrastructure.

#### Scenario: Visitor browses demo resources
- **WHEN** the demo administrator opens pages for nodes, cases, tasks, users, channels, topics, or statistics
- **THEN** the system displays seeded demo data
- **AND** restarts later do not retain any changes made during the demo

### Requirement: Server-side readonly enforcement
Live demo mode SHALL reject API requests that create, update, delete, execute, register, restart, retry, cancel, or otherwise mutate demo state, except the demo login and logout operations.

#### Scenario: Mutation attempt is blocked
- **WHEN** an authenticated visitor sends a `POST`, `PUT`, `PATCH`, or `DELETE` request to a business or admin API
- **THEN** the server responds with a readonly live demo error
- **AND** no demo data changes

#### Scenario: Operational side effect is blocked
- **WHEN** an authenticated visitor calls a restart, retry, cancel, or token-rotation API
- **THEN** the server rejects the request before invoking the operation
- **AND** the demo runtime remains available with unchanged demo state

### Requirement: Setup and credential changes blocked
Live demo mode SHALL hide the settings menu entry and SHALL reject setup, configuration, registration, and password-change APIs at the server.

#### Scenario: Settings entry is unavailable
- **WHEN** the demo administrator views the admin navigation
- **THEN** the settings menu entry is not rendered

#### Scenario: Sensitive API is unavailable
- **WHEN** a visitor calls setup, configuration, registration, or password-change APIs
- **THEN** the server responds with a readonly live demo error
- **AND** the request does not modify configuration, credentials, or account state
