DROP TABLE IF EXISTS schedule;

CREATE TABLE IF NOT EXISTS schedule (
    schedule_index STRING NOT NULL,
    schedule_type STRING NOT NULL,
    schedule_group INT NOT NULL,
    day STRING NOT NULL,
    timeText STRING NOT NULL,
    timeStart STRING NOT NULL,
    timeEnd STRING NOT NULL,
    venue STRING NOT NULL,
    remark STRING
);

DROP TABLE IF EXISTS subject;

CREATE TABLE IF NOT EXISTS subject (
    id STRING NOT NULL,
    title STRING NOT NULL,
    rawAU STRING NOT NULL
);
