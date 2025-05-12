# Habit Tracker Web App 🌱  
📝 Overview

A web-based habit tracking application that helps users build and maintain daily and weekly habits. Built with Go (Golang) for the backend and HTML/HTMX for a responsive frontend.
✨ Features

    Signup and Login

    Track daily and weekly habits

    Mark habits as completed/skipped

    Progress tracking with visual indicators 

    Create, edit, and delete habits

    Simple, intuitive interface 

🛠️ Technologies

    Backend: Go (Golang)

    Frontend: HTML5, HTMX, CSS

    Database: PostgreSQL

    Templating: Go html/template  

Set up your environment  
Create a .envrc or set your environment variable:  
```export TRACKER_DB_DSN=postgres://your_user:your_pass@localhost/tracker?sslmode=disable  ```

🗄️ Database Schema (PostgreSQL)

**Note:** An image of this can be found in the folder _DB-Schema_

Run the migrations in /migrations/ or use your own tool.

Tables include:

    habits

    habit_entries

    users