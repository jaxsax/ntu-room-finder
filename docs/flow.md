URL0=https://wish.wis.ntu.edu.sg/webexe/owa/aus_schedule.main
URL1=https://wish.wis.ntu.edu.sg/webexe/owa/AUS_SCHEDULE.main_display1

# Goroutines design

# File structure

TODAY=2018-09-13

## $TODAY/main

This is the page that was fetched to retrieve the latest academic semester and course list

## $TODAY/mapping.json

This json file maps a course hash to its real name

## $TODAY/<hash>

This folder contains the hash of the parameters which uniquely identify a course.
This folder would contain all schedules that are parsed from $URL1

## $TODAY/<hash>/mapping.json

This json file maps the hash of the schedules to its real name

# Component: Crawler

## Step 1 .. Figuring out correct acadsem value

1. GET $URL0
1. Figure out the latest acadsem to use

## Step 2 .. Setting up queue with course parameters

1. GET $URL0 w/ correct acadsem
1. Using the parameters from step 1, pull out all course parameters

## Step 3 .. Iterate over queue, retrieving course details

1. POST $URL1 w/ correct parameters
1. Parse contents of this page into a Go type
1. Write those contents into a database


## Overview

(acadsem) -> (acadmsem_courses) -> worker:(acadsem_courses_details) -> worker:(insert_database)

Queues:
1. Parse courses queue
1. Insert into database queue

# Component: Figure-outer
