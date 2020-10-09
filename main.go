package main

import (
	"fmt"
	"os"
	"errors"
  "io/ioutil"
	"os/signal"
	"syscall"
  "strings"
  //"strconv"
  "encoding/json"

	"github.com/bwmarrin/discordgo"
)

type RecipeItem struct {
  Id int
  Name string
  Quantity int
}
type recipe = []RecipeItem

var name2id = make(map[string]int)
var id2name = make(map[int]string)
var recipes = make(map[string]recipe)

func buildRecipe(id int) (map[int]int, error) {
  recipe, present := recipes[fmt.Sprintf("%d", id)] // ???
  if !present {
    return nil, errors.New("No recipe found.")
  }
  recipeQty := make(map[int]int)
  for _, recipeItem := range recipe {
    subRecipeRaw, hasSub := recipes[fmt.Sprintf("%d", recipeItem.Id)]
    if hasSub && len(subRecipeRaw) > 0 {
      subRecipe, err := buildRecipe(recipeItem.Id)
      if err == nil {
        for subId, subQty := range subRecipe {
          curQty, present := recipeQty[subId]
          if !present {
            curQty = 0
          }
          curQty += subQty * recipeItem.Quantity
          recipeQty[subId] = curQty
        }
      }
    } else {
      recipeQty[recipeItem.Id] = recipeItem.Quantity
    }
  }
  return recipeQty, nil
}

func quantities2String(qties map[int]int) string {
  var sb strings.Builder
  for id, qty := range qties {
    fmt.Fprintf(&sb, "%dx ", qty)

    if name, present := id2name[id]; present {
      sb.WriteString(name)
    } else {
      sb.WriteString("?")
    }
    sb.WriteString("\n")
  }
  return sb.String()
}

func loadNames() error {
  name2idFile, err := os.Open("name_to_id.json")
  if err != nil {
    return err
  }
  defer name2idFile.Close()
  name2idBytes, err := ioutil.ReadAll(name2idFile)
  if err != nil {
    return err
  }

  json.Unmarshal(name2idBytes, &name2id)
  for name, id := range name2id {
    id2name[id] = name
  }
  return nil
}

func loadRecipes() error {
  recipeFile, err := os.Open("recipe_by_id.json")
  if err != nil {
    return err
  }
  defer recipeFile.Close()
  recipeBytes, err := ioutil.ReadAll(recipeFile)
  if err != nil {
    return err
  }

  json.Unmarshal(recipeBytes, &recipes)
  return nil
}

func main() {
  err := loadNames()
  if err != nil {
		fmt.Println("Error loading name_to_id json,", err)
		return
  }

  err = loadRecipes()
  if err != nil {
		fmt.Println("Error loading recipes json,", err)
		return
  }

	dg, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}
  defer dg.Close()

	dg.AddHandler(messageCreate)
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

  args := strings.SplitN(m.Content, " ", 2)
  if len(args) < 2 {
    return
  }

  // id, err := strconv.atoi(args[1])
  // if err != nil{
  //   s.ChannelMessageSend(m.ChannelID, "Cannot parse id.")
  //   return
  // }

	if args[0] == "!id" {
    id, present := name2id[args[1]]
    if !present {
      s.ChannelMessageSend(m.ChannelID, "Item not found.")
      return
    }

    s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Item ID is: %d", id))
	}

	if args[0] == "!recipe" {
    id, present := name2id[args[1]]
    if !present {
      s.ChannelMessageSend(m.ChannelID, "Item not found.")
      return
    }

    qties, err := buildRecipe(id)
    if err != nil {
      s.ChannelMessageSend(m.ChannelID, "Unable to build recipe.")
      return
    }
    s.ChannelMessageSend(m.ChannelID, quantities2String(qties))
	}
}

