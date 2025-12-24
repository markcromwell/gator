-- +goose Up
/*
A post is a single entry from a feed. It should have:

id - a unique identifier for the post
created_at - the time the record was created
updated_at - the time the record was last updated
title - the title of the post
url - the URL of the post (this should be unique)
description - the description of the post
published_at - the time the post was published
feed_id - the ID of the feed that the post came from
*/

CREATE TABLE 
posts (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    title VARCHAR(255) NOT NULL,
    url VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    published_at TIMESTAMP NOT NULL,
    feed_id UUID NOT NULL,
    FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE posts;