# dumbledodo.dev

The source of my portfolio site, [dumbledodo.dev](https://dumbledodo.dev).

It is a Go server that uses only the standard library, with htmx. Eerything is embedded into one binary.

## Layout

    main.go            server, handlers and data loading
    github.go          GitHub numbers for the "On GitHub" section
    heatmap.go         commit heat map
    avatars.go         reviewer pictures
    templates/         page and htmx fragments
    static/            styles, script, vendored htmx, fonts and images
    data/              the content, as JSON
