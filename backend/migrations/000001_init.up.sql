CREATE TABLE "user" (
    id_pk SERIAL PRIMARY KEY,
    login VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(50) CHECK (role IN ('admin', 'developer', 'qa')),
    ver INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE organizations (
    id_pk SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

CREATE TABLE org_member (
    org_id_fk INTEGER REFERENCES organizations(id_pk) ON DELETE CASCADE,
    user_id_fk INTEGER REFERENCES "user"(id_pk) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    PRIMARY KEY (org_id_fk, user_id_fk)
);

CREATE TABLE projects (
    id_pk SERIAL PRIMARY KEY,
    org_id_fk INTEGER REFERENCES organizations(id_pk) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL
);

CREATE TABLE project_member (
    project_id_fk INTEGER REFERENCES projects(id_pk) ON DELETE CASCADE,
    user_id_fk INTEGER REFERENCES "user"(id_pk) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('pm', 'dev', 'qa', 'viewer')),
    position VARCHAR(100),
    PRIMARY KEY (project_id_fk, user_id_fk)
);

CREATE TABLE task (
    id_pk SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    owner_id_fk INTEGER REFERENCES "user"(id_pk) ON DELETE CASCADE,
    project_id_fk INTEGER REFERENCES projects(id_pk) ON DELETE CASCADE,
    status VARCHAR(50) CHECK (status IN (
        'New', 'Open', 'In Progress', 'Fixed', 'Ready for Retest', 
        'Verified', 'Reopened', 'Rejected', 'Can''t Reproduce'
    )),
    severity VARCHAR(50) CHECK (severity IN ('Blocker', 'Critical', 'Major', 'Minor')),
    priority VARCHAR(50) CHECK (priority IN ('High', 'Medium', 'Low')),
    os VARCHAR(200),
    version_product VARCHAR(50),
    playback_description TEXT,
    expected_result TEXT,
    actual_result TEXT,
    assigned_to_fk INTEGER REFERENCES "user"(id_pk) ON DELETE SET NULL,
    assigned_time TIMESTAMPTZ,
    passed_by_fk INTEGER REFERENCES "user"(id_pk) ON DELETE SET NULL,
    passed_time TIMESTAMPTZ,
    accepted_by_fk INTEGER REFERENCES "user"(id_pk) ON DELETE SET NULL,
    accepted_time TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE bugs (
    id_pk SERIAL PRIMARY KEY,
    task_id_fk INTEGER NOT NULL REFERENCES task(id_pk) ON DELETE CASCADE,
    created_by_fk INTEGER REFERENCES "user"(id_pk) ON DELETE SET NULL,
    assigned_to_fk INTEGER REFERENCES "user"(id_pk) ON DELETE SET NULL,
    passed_by_fk INTEGER REFERENCES "user"(id_pk) ON DELETE SET NULL,
    accepted_by_fk INTEGER REFERENCES "user"(id_pk) ON DELETE SET NULL,
    severity VARCHAR(50) CHECK (severity IN ('Blocker', 'Critical', 'Major', 'Minor')),
    priority VARCHAR(50) CHECK (priority IN ('High', 'Medium', 'Low')),
    status VARCHAR(50) DEFAULT 'Open' CHECK (status IN (
        'New', 'Open', 'In Progress', 'Fixed', 'Ready for Retest',
        'Verified', 'Reopened', 'Rejected', 'Can''t Reproduce'
    )),
    description TEXT,
    playback_description TEXT,
    expected_result TEXT,
    actual_result TEXT,
    version_product VARCHAR(50),
    os VARCHAR(200),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    assigned_time TIMESTAMPTZ,
    passed_time TIMESTAMPTZ,
    accepted_time TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_bugs_task_id ON bugs(task_id_fk);

CREATE TABLE bug_comment (
    id_pk SERIAL PRIMARY KEY,
    bug_id_fk INTEGER NOT NULL REFERENCES bugs(id_pk) ON DELETE CASCADE,
    user_id_fk INTEGER REFERENCES "user"(id_pk) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_bug_comment_bug_id ON bug_comment(bug_id_fk);

-- Photo metadata (MinIO object storage). Actual bytes are stored in S3-compatible storage.
CREATE TABLE photo (
    id_pk SERIAL PRIMARY KEY,
    entity_type VARCHAR(10) NOT NULL CHECK (entity_type IN ('task', 'bug')),
    entity_id INTEGER NOT NULL,
    object_key TEXT NOT NULL,
    url TEXT NOT NULL,
    uploaded_by_fk INTEGER REFERENCES "user"(id_pk) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_photo_entity ON photo(entity_type, entity_id);
CREATE INDEX idx_photo_uploader ON photo(uploaded_by_fk);

CREATE TABLE task_comment (
    id_pk SERIAL PRIMARY KEY,
    task_id_fk INTEGER REFERENCES task(id_pk) ON DELETE CASCADE,
    user_id_fk INTEGER REFERENCES "user"(id_pk) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE task_audit (
    id_pk SERIAL PRIMARY KEY,
    task_id_fk INTEGER REFERENCES task(id_pk) ON DELETE CASCADE,
    user_id_fk INTEGER REFERENCES "user"(id_pk) ON DELETE CASCADE,
    field VARCHAR(100) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION audit_task_changes()
RETURNS TRIGGER AS $$
DECLARE
    curr_user_id INTEGER;
BEGIN
    -- Пытаемся получить ID пользователя из настройки сессии (установим его позже из Go)
    BEGIN
        curr_user_id := current_setting('app.current_user_id')::integer;
    EXCEPTION WHEN OTHERS THEN
        curr_user_id := NULL; -- Если не установлен, будет NULL
    END;

    -- Проверяем основные поля на изменения
    IF (OLD.title IS DISTINCT FROM NEW.title) THEN
        INSERT INTO task_audit (task_id_fk, user_id_fk, field, old_value, new_value)
        VALUES (NEW.id_pk, curr_user_id, 'title', OLD.title, NEW.title);
    END IF;

    IF (OLD.description IS DISTINCT FROM NEW.description) THEN
        INSERT INTO task_audit (task_id_fk, user_id_fk, field, old_value, new_value)
        VALUES (NEW.id_pk, curr_user_id, 'description', OLD.description, NEW.description);
    END IF;

    IF (OLD.status IS DISTINCT FROM NEW.status) THEN
        INSERT INTO task_audit (task_id_fk, user_id_fk, field, old_value, new_value)
        VALUES (NEW.id_pk, curr_user_id, 'status', OLD.status, NEW.status);
    END IF;

    IF (OLD.severity IS DISTINCT FROM NEW.severity) THEN
        INSERT INTO task_audit (task_id_fk, user_id_fk, field, old_value, new_value)
        VALUES (NEW.id_pk, curr_user_id, 'severity', OLD.severity, NEW.severity);
    END IF;

    IF (OLD.priority IS DISTINCT FROM NEW.priority) THEN
        INSERT INTO task_audit (task_id_fk, user_id_fk, field, old_value, new_value)
        VALUES (NEW.id_pk, curr_user_id, 'priority', OLD.priority, NEW.priority);
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_task_audit
AFTER UPDATE ON task
FOR EACH ROW
EXECUTE FUNCTION audit_task_changes();


CREATE TABLE bug_relation (
    id_pk SERIAL PRIMARY KEY,
    bug_id_a_fk INTEGER REFERENCES bugs(id_pk) ON DELETE CASCADE,
    bug_id_b_fk INTEGER REFERENCES bugs(id_pk) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('related', 'duplicate', 'blocks')),
    CONSTRAINT unique_relation UNIQUE (bug_id_a_fk, bug_id_b_fk)
);

CREATE TABLE bug_tag (
    id_pk SERIAL PRIMARY KEY,
    bug_id_fk INTEGER REFERENCES bugs(id_pk) ON DELETE CASCADE,
    tag VARCHAR(100) NOT NULL
);

CREATE TABLE template (
    id_pk SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    created_by_fk INTEGER REFERENCES "user"(id_pk) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE chat_thread (
    id_pk SERIAL PRIMARY KEY,
    scope VARCHAR(20) NOT NULL CHECK (scope IN ('org', 'project', 'dm')),
    org_id_fk INTEGER REFERENCES organizations(id_pk) ON DELETE CASCADE,
    project_id_fk INTEGER REFERENCES projects(id_pk) ON DELETE CASCADE,
    dm_user_a_fk INTEGER REFERENCES "user"(id_pk) ON DELETE CASCADE,
    dm_user_b_fk INTEGER REFERENCES "user"(id_pk) ON DELETE CASCADE,
    created_by_fk INTEGER REFERENCES "user"(id_pk) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Проверка: ID_A всегда должен быть меньше ID_B (для уникальности пар в DM)
    CONSTRAINT check_dm_order CHECK (
        (scope = 'dm' AND dm_user_a_fk < dm_user_b_fk) OR scope != 'dm'
    )
);

CREATE TABLE chat_message (
    id_pk SERIAL PRIMARY KEY,
    thread_id_fk INTEGER REFERENCES chat_thread(id_pk) ON DELETE CASCADE,
    user_id_fk INTEGER REFERENCES "user"(id_pk) ON DELETE SET NULL,
    body TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    edited_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE chat_read_state (
    thread_id_fk INTEGER REFERENCES chat_thread(id_pk) ON DELETE CASCADE,
    user_id_fk INTEGER REFERENCES "user"(id_pk) ON DELETE CASCADE,
    last_read_message_id INTEGER REFERENCES chat_message(id_pk) ON DELETE SET NULL,
    last_read_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (thread_id_fk, user_id_fk)
);

CREATE TABLE chat_typing_state (
    thread_id_fk INTEGER REFERENCES chat_thread(id_pk) ON DELETE CASCADE,
    user_id_fk INTEGER REFERENCES "user"(id_pk) ON DELETE CASCADE,
    is_typing BOOLEAN DEFAULT FALSE,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (thread_id_fk, user_id_fk)
);
