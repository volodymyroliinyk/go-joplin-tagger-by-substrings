# go-joplin-tagger-by-substrings

# Joplin Tagger By Substring(s) (Golang)

- This is a simple [Go](https://go.dev/)script that tags your [Joplin](https://joplinapp.org/) notes with a specified
  tag if the text of the note matches the string or strings specified.

## Installation and configuration

1. Setting up the Joplin API
    - Before running the script, you must activate the [Joplin Web Clipper](https://joplinapp.org/help/apps/clipper/)
      API and obtain your token:
        - Open the [Joplin Desktop](https://joplinapp.org/help/install/#desktop-applications) application.
        - Go to Tools -> Options -> Web Clipper.
        - Enable the "Enable Web Clipper Service" option.
        - Copy the Authorization token from there.
        - Make sure Joplin Desktop is running when you run the script.
2. Go script configuration
    - Open the main.go file and replace the stub with your real token:
    ```
    const JOPLIN_API_BASE = "http://localhost:41184"
    const JOPLIN_TOKEN = "YOUR_COPIED_API_TOKEN"
    ```
3. Launch
    - Make sure you have Go (Golang) installed.
    - Save the file as main.go.
    - Open a terminal in the directory with the file.
    - Run the script: \
      ```go run main.go --tag_name="<existing or new tag name>" --contains_substring="<substring 1>" --contains_substring="<substring 2>";```
      ```go run main.go --tag_name="<existing or new tag name>" --contains_substring="<substring>"";```

TODO:

- unit testing