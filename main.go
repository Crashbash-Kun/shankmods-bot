package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
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
    dg.AddHandler(voiceStateUpdate)
    dg.AddHandler(messageReactionAdd)

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
    Updates the responses value that's held in memory with those stored in JSON on github
*/
func updateResponses(){
    for {
        err, body := getResponsesJSON()
        if err != nil || len(string(body)) == 0 {
        } else {
            err := json.Unmarshal(body, &responses)
            if err != nil {
                fmt.Println("JSON isn't valid")
            }
        }

        time.Sleep(150 * time.Second)
    }
}

/*
    Handles making a GET request to github and returns the json body of said request
*/
func getResponsesJSON() (err error, body []byte) {
    url := "https://raw.githubusercontent.com/Crashbash-Kun/shankmods-bot/master/responses.json"
    client := http.Client{
        Timeout: time.Second * 5,
    }

    req, err := http.NewRequest(http.MethodGet, url, nil)
    if err != nil {
        return err, nil
    }

    res, getErr := client.Do(req)
    if getErr != nil {
        return err, nil
    }

    defer res.Body.Close()

    body, readErr := ioutil.ReadAll(res.Body)
    if readErr != nil {
        return err, nil
    }
    return nil, body
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

/*
    Handler that activates when a user's voice-status changes.
    If the user is joining a voice channel, they are given a role. If they
    are disconnecting from VC then the role is removed.
*/
func voiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
    vcRole  := "715712953990512780" // Prod
    guildID := "474318493081403420" // Prod
    //guildID := "705153111412703324" // Test
    //vcRole  := "715712631541071924" // Test

    // Give role if the VC channel is in the server
    if    v.ChannelID != "" && v.GuildID == guildID {
        err := s.GuildMemberRoleAdd(guildID, v.UserID, vcRole)
        if err != nil {
            fmt.Println("Adding vc role:", err)
        }

    // Remove role if they join another server's VC chat, or disconnect from VC in the server
    } else if v.GuildID == guildID {
        err := s.GuildMemberRoleRemove(guildID, v.UserID, vcRole)
        if err != nil {
            fmt.Println("Remove vc role:", err)
        }
    }
}

/*

*/
func messageReactionAdd(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
    shankID := "265697070093041666"
    submissionID := "560905346772762627"
    fridgeChatID := "715819525517344818"
    emojiID := "🧲"
    emojiIDDone :="❤️"
    requiredLikes := 5

    // Check this is in the right channel
    if m.ChannelID != submissionID {
        return
    }

    // Check not already posted
    reactors, err := s.MessageReactions(m.ChannelID, m.MessageID, emojiIDDone, 100)
    if err != nil {
        fmt.Println("posted-check react check", err)
        return
    }

    bot, err := s.User("@me")
    if err != nil {
        fmt.Println("@me", err)
        return
    }
    for _, reactor := range reactors {
        if reactor.ID == bot.ID {
            return
        }
    }

    reactors, err = s.MessageReactions(m.ChannelID, m.MessageID, emojiID, 100)
    if err != nil {
        fmt.Println("message reactions", err)
        return
    }

    shouldBePosted := len(reactors) >= requiredLikes

    if !shouldBePosted {
        for _, reactor := range reactors {
            shouldBePosted = shouldBePosted || reactor.ID == shankID
        }
    }

    if shouldBePosted {
        message, err := s.ChannelMessage(m.ChannelID, m.MessageID)
        if err != nil {
            return
        }

        // Construct structs to build our really weird embed message
        embedAuthor := &discordgo.MessageEmbedAuthor{}
        embedAuthor.Name = message.Author.Username
        embedAuthor.IconURL = message.Author.AvatarURL("128")

        source := &discordgo.MessageEmbedField{}
        source.Name = "Original Post"
        source.Value = "[Click me](https://discordapp.com/channels/474318493081403420/" + m.ChannelID + "/" + m.MessageID + ")"
        source.Inline = true

        fields := make([]*discordgo.MessageEmbedField, 1, 10)
        fields[0] = source

        embed := &discordgo.MessageEmbed{}
        embed.Description = message.Content
        embed.Author = embedAuthor
        embed.Fields = fields

        if len(message.Embeds) > 0 {
            ogEmbed:= message.Embeds[0]
            embed.URL = ogEmbed.URL
            embed.Image = ogEmbed.Image
            embed.Thumbnail = ogEmbed.Thumbnail
            embed.Video = ogEmbed.Video
            embed.Provider = ogEmbed.Provider

            image := &discordgo.MessageEmbedImage{}
            image.URL = ogEmbed.Thumbnail.URL
            image.ProxyURL = ogEmbed.Thumbnail.ProxyURL
            image.Width = ogEmbed.Thumbnail.Width
            image.Height = ogEmbed.Thumbnail.Height
            embed.Image = image
        }

        if len(message.Attachments) > 0 {
            ogAttachment := message.Attachments[0]

            image := &discordgo.MessageEmbedImage{}
            image.URL = ogAttachment.URL
            image.ProxyURL = ogAttachment.ProxyURL
            image.Width = ogAttachment.Width
            image.Height = ogAttachment.Height

            embed.Image = image
        }

        s.ChannelMessageSendEmbed(fridgeChatID, embed)
        s.MessageReactionAdd(m.ChannelID, m.MessageID, emojiIDDone)
    }

}
