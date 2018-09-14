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
    schedule_index NOT NULL,
    title STRING NOT NULL,
    rawAU STRING NOT NULL
);

DROP VIEW IF EXISTS schedule_d;
CREATE VIEW schedule_d AS SELECT DISTINCT * FROM schedule;

DROP VIEW IF EXISTS subject_d;
CREATE VIEW subject_d AS SELECT DISTINCT * FROM subject;

DROP VIEW IF EXISTS schedule_day;
CREATE VIEW schedule_day AS
    SELECT  schedule_index, schedule_type, schedule_group, day,
            (case
                when day = 'MON' then 1
                when day = 'TUE' then 2
                when day = 'WED' then 3
                when day = 'THU' then 4
                when day = 'FRI' then 5
                when day = 'SAT' then 6
                when day = 'SUN' then 7
            end) AS day_number,
            timeText, timeStart, timeEnd, venue, remark
    FROM schedule_d;

DROP VIEW IF EXISTS tutorial_rooms;
CREATE VIEW tutorial_rooms AS
    SELECT * FROM schedule_day WHERE schedule_type = 'TUT';
