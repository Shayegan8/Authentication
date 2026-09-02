CREATE TABLE IF NOT EXISTS users(
    userid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE, username VARCHAR(50) UNIQUE,
    password BYTEA, refreshToken VARCHAR(64), timestamp NUMERIC DEFAULT EXTRACT(EPOCH FROM NOW())
);

CREATE INDEX email_idx ON users(email);

CREATE OR REPLACE FUNCTION checker() RETURNS trigger AS $$
DECLARE
    timen numeric;
    timenjerk numeric;
    BEGIN
        timen := EXTRACT(EPOCH FROM NOW());
        timenjerk := timen - OLD.timestamp;
        IF timenjerk > 2419200 THEN
            -- Well in here i really dont see any reasoning why i should remove something with update from users, exception is fine
            RAISE EXCEPTION 'You don''t have this refresh token anymore, login again';
        END IF;
    END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER checker BEFORE INSERT OR UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION checker();

CREATE TABLE IF NOT EXISTS posts(
    postid UUID PRIMARY KEY DEFAULT gen_random_uuid(), userid UUID REFERENCES users(userid) NOT NULL,
    title VARCHAR(60) NOT NULL, info VARCHAR(255) NOT NULL, body TEXT NOT NULL, timestamp NUMERIC DEFAULT EXTRACT(EPOCH FROM NOW())
);

CREATE TABLE IF NOT EXISTS admins(
    adminid UUID REFERENCES users(userid)
);

CREATE OR REPLACE PROCEDURE insert_post(
    userid_ UUID, 
    title_ VARCHAR(80), 
    info_ TEXT, 
    body_ TEXT
) 
LANGUAGE plpgsql 
AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM admins WHERE userid=userid_) THEN
        UPDATE users SET timestamp = EXTRACT(EPOCH FROM NOW()) WHERE userid = userid_;
        INSERT INTO posts(title, info, body) VALUES (title_, info_, body_);
    ELSE
        RAISE EXCEPTION 'You are not admin'; -- in case user wasnt admin
    END IF;
END;
$$;

CREATE TABLE IF NOT EXISTS comments(
    commentid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    postid UUID REFERENCES posts(postid) NOT NULL, userid UUID REFERENCES users(userid) NOT NULL,
    body VARCHAR(104000) NOT NULL, timestamp NUMERIC DEFAULT EXTRACT(EPOCH FROM NOW())
);

CREATE TABLE IF NOT EXISTS replies(
    replyid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    commentid UUID REFERENCES comments(commentid) NOT NULL,
    postid UUID REFERENCES posts(postid) NOT NULL, userid UUID REFERENCES users(userid) NOT NULL,
    body VARCHAR(104000) NOT NULL, timestamp NUMERIC DEFAULT EXTRACT(EPOCH FROM NOW())
);

CREATE OR REPLACE PROCEDURE insert_comment(
    postid_ UUID,
    userid_ UUID,
    body_ TEXT
) LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM userid WHERE userid=userid_) THEN
        IF EXISTS (SELECT 1 FROM posts WHERE postid=postid_) THEN
            UPDATE users SET timestamp = EXTRACT(EPOCH FROM NOW()) WHERE userid = userid_;
            INSERT INTO comments(postid, userid, body) VALUES(postid_, userid_, body_) RETURNING commentid;
        ELSE
            RAISE EXCEPTION 'Post dosen''t exist';
        END IF;
    ELSE
        RAISE EXCEPTION 'User dosen''t exist';
    END IF;
END;
$$;

CREATE OR REPLACE PROCEDURE insert_reply(
    postid_ UUID,
    userid_ UUID,
    commentid_ UUID,
    body_ TEXT
) LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM userid WHERE userid=userid_) THEN
        IF EXISTS (SELECT 1 FROM posts WHERE postid=postid_) THEN
            IF EXISTS (SELECT 1 FROM comments WHERE commentid=commentid_) THEN
                UPDATE users SET timestamp = EXTRACT(EPOCH FROM NOW()) WHERE userid = userid_;
                INSERT INTO replies(postid, userid, commentid, body) VALUES (postid_, userid_, commentid_, body_) RETURNING replyid;
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
$$;