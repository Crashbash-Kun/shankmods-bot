package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "os/signal"
    "strconv"
    "strings"
    "syscall"
    "time"

    "github.com/bwmarrin/discordgo"
)

// Variables used for command line parameters
var (
    Token string
)

// Stores our responses. These will be updated every so often so we don't have to restart the bot to make changes
var responses map[string] string

/*
    Initialization code for the bot. Pulls the required API key from commandline arguments.
*/
func init() {

    flag.StringVar(&Token, "t", "", "Bot Token")
    flag.Parse()
}

/*
    Main function that handles creation of discord session, callbacks, and handlers.
*/
func main() {

    // Create a new Discord session with token.
    dg, err := discordgo.New("Bot " + Token)
    if err != nil {
        fmt.Println("error creating Discord session,", err)
        return
    }

    // register callback for new messages
    dg.AddHandler(messageCreate)

    // Open a websocket, listen
    err = dg.Open()
    if err != nil {
        fmt.Println("error opening connection,", err)
        return
    }

    fmt.Println("Bot running.")

    // Start additional go routines
    go setStatus(dg)
    go updateResponses()

    // Set custom system call handlers for user interrupt and wait for one to occurr.
    sc := make(chan os.Signal, 1)
    signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
    <-sc
    dg.Close()
}

/*
    Everytime a new message is made in a channel the bot has access to, this
    function will be called in regards to that message
*/
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

    // Ignore all messages created by the bot itself
    if m.Author.ID == s.State.User.ID {
        return
    }

    if m.Content == "!users" {
        s.ChannelMessageSend(m.ChannelID, "The bot has " + strconv.Itoa(countUsers(s)) + " users.\n")
        return
    }
    
    // Only look for a command if message starts with a bang
    if len(m.Content) > 0 && strings.HasPrefix(m.Content, "!") {
        
        // Get first word, use that to determine what to do
        var command string
        command = strings.TrimLeft(m.Content, " ")
        command = strings.TrimPrefix(command, "!")
        command = strings.ToLower(command)
        
        //fmt.Println(command, responses[command])
        
        if responses[command] != "" {
            s.ChannelMessageSend(m.ChannelID, string(responses[command]))
        }
    }
}

/*
    Downloads JSON file from github and updates value in memory
*/
func updateResponses(){
    for {
        url := "https://raw.githubusercontent.com/Crashbash-Kun/shankmods-bot/master/responses.json"

        client := http.Client{
            Timeout: time.Second * 60,
        }

        req, err := http.NewRequest(http.MethodGet, url, nil)
        if err != nil {
            log.Fatal(err)
            return
        }

        res, getErr := client.Do(req)
        if getErr != nil {
            log.Fatal(getErr)
            return
        }

        defer res.Body.Close()
        
	body, readErr := ioutil.ReadAll(res.Body)
        if readErr != nil {
            log.Fatal(readErr)
            return
        }

        jsonErr := json.Unmarshal(body, &responses)
        if jsonErr != nil {
            log.Fatal(jsonErr)
            return
        }
        
        fmt.Println(responses)
        
        time.Sleep(150 * time.Second)
    }
}

/*
    Counts the 'total number' of users that are exposed to that bot via servers it is in.
    This is not a count of unique users and includes duplicate users that appear in different servers.
*/
func countUsers(s *discordgo.Session) int {
    count := 0
    for _, guild := range s.State.Guilds {
        count += guild.MemberCount
    }

    return count
}

/*
    Sets the message seen in discord when users hover over the bot.
    This is repeated ocassionally as not doing so rsults in the status eventually dissapearing.
*/
func setStatus(s *discordgo.Session) {
    for {
        s.UpdateStatus(-1, "Modding it up")
        time.Sleep(5 * time.Minute)
    }
}
