CREATE TABLE IF NOT EXISTS users(
    userid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE, username VARCHAR(50) UNIQUE,
    password BYTEA
);

CREATE TABLE IF NOT EXISTS sessions(
    sessionid UUID UNIQUE DEFAULT gen_random_uuid(),
    userid UUID NOT NULL REFERENCES users(userid) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL REFERENCES users(email) ON DELETE CASCADE,
    refreshToken BYTEA UNIQUE, timestamp NUMERIC DEFAULT EXTRACT(EPOCH FROM NOW())
);

CREATE INDEX ON users(email);
CREATE INDEX ON users(username);
CREATE INDEX ON sessions(sessionid);
CREATE INDEX ON sessions(userid);
CREATE INDEX ON sessions(email);

CREATE OR REPLACE FUNCTION checker() RETURNS trigger AS $$
DECLARE
    timen numeric;
    timenjerk numeric;
    BEGIN
        timen := EXTRACT(EPOCH FROM NOW());
        timenjerk := timen - OLD.timestamp;
        IF timenjerk > 2419200 THEN
            -- DELETE FROM sessions WHERE sessionid = OLD.sessionid; we do this delete in comments and replies apis
            RAISE EXCEPTION 'You don''t have this refresh token anymore, login again';
        END IF;
        RETURN NEW;
    END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER checker BEFORE UPDATE ON sessions
    FOR EACH ROW EXECUTE FUNCTION checker();

CREATE TABLE IF NOT EXISTS posts(
    postid UUID PRIMARY KEY DEFAULT gen_random_uuid(), userid UUID NOT NULL REFERENCES users(userid) ON DELETE CASCADE,
    title VARCHAR(60) NOT NULL, info VARCHAR(255) NOT NULL, body TEXT NOT NULL, timestamp NUMERIC DEFAULT EXTRACT(EPOCH FROM NOW())
);

CREATE TABLE IF NOT EXISTS admins(
    adminid UUID NOT NULL REFERENCES users(userid) ON DELETE CASCADE
);

CREATE OR REPLACE FUNCTION check_userious(_email VARCHAR(255))
RETURNS TABLE (userid UUID, password BYTEA)
AS
$$
DECLARE
    nsessions NUMERIC;
BEGIN
    SELECT COUNT(*) INTO nsessions FROM sessions WHERE email = _email;

    if nsessions == 5 THEN
        RAISE EXCEPTION 'You can''t have more than 5 device here';
    END IF;

    RETURN QUERY SELECT u.userid, u.password FROM users u WHERE u.email = _email;
END;
$$
LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS comments(
    commentid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    postid UUID NOT NULL REFERENCES posts(postid) ON DELETE CASCADE, userid UUID NOT NULL REFERENCES users(userid) ON DELETE CASCADE,
    body VARCHAR(104000) NOT NULL, timestamp NUMERIC DEFAULT EXTRACT(EPOCH FROM NOW())
);

CREATE TABLE IF NOT EXISTS replies(
    replyid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    commentid UUID NOT NULL REFERENCES comments(commentid) ON DELETE CASCADE,
    postid UUID NOT NULL REFERENCES posts(postid) ON DELETE CASCADE, userid UUID NOT NULL REFERENCES users(userid) ON DELETE CASCADE,
    body VARCHAR(104000) NOT NULL, timestamp NUMERIC DEFAULT EXTRACT(EPOCH FROM NOW())
);

CREATE OR REPLACE FUNCTION insert_comment(
    postid_ UUID,
    userid_ UUID,
    sessionid_ UUID,
    body_ TEXT
) RETURNS UUID
AS $$
DECLARE result UUID;
BEGIN
    IF EXISTS (SELECT 1 FROM users WHERE userid=userid_) THEN
        IF EXISTS (SELECT 1 FROM posts WHERE postid=postid_) THEN
            IF EXISTS (SELECT 1 FROM sessions WHERE sessionid=sessionid_) THEN
                UPDATE sessions SET timestamp = EXTRACT(EPOCH FROM NOW()) WHERE sessionid = sessionid_;
                INSERT INTO comments(postid, userid, body) VALUES(postid_, userid_, body_) RETURNING commentid INTO result;
                RETURN result;
            ELSE
                RAISE EXCEPTION 'Session dosen''t exist';
            END IF;
        ELSE
            RAISE EXCEPTION 'Post dosen''t exist';
        END IF;
    ELSE
        RAISE EXCEPTION 'User dosen''t exist';
    END IF;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION insert_reply(
    postid_ UUID,
    userid_ UUID,
    commentid_ UUID,
    sessionid_ UUID,
    body_ TEXT
) RETURNS UUID
AS $$
DECLARE result UUID;
BEGIN
    IF EXISTS (SELECT 1 FROM users WHERE userid=userid_) THEN
        IF EXISTS (SELECT 1 FROM posts WHERE postid=postid_) THEN
            IF EXISTS (SELECT 1 FROM comments WHERE commentid=commentid_) THEN
                IF EXISTS (SELECT 1 FROM sessions WHERE sessionid=sessionid_) THEN
                    UPDATE sessions SET timestamp = EXTRACT(EPOCH FROM NOW()) WHERE sessionid = sessionid_;
                    INSERT INTO replies(postid, userid, commentid, body) VALUES (postid_, userid_, commentid_, body_) RETURNING replyid INTO result;
                    RETURN result; 
                ELSE
                    RAISE EXCEPTION 'session dosent''t exist';
                END IF;
            ELSE
                RAISE EXCEPTION 'Comment dosent''t exist';
            END IF;
        ELSE
            RAISE EXCEPTION 'Post dosen''t exist';
        END IF;
    ELSE
        RAISE EXCEPTION 'User dosen''t exist';
    END IF;
END;
$$
LANGUAGE plpgsql;