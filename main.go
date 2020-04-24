package main

import (
	"context"
	"github.com/lifei6671/clashx-convert/server"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "clashx-convert"
	app.Usage = "Convert vmess subscription format to clashx."
	app.Version = "0.1"
	app.Commands = []*cli.Command{
		&cli.Command{
			Name:  "run",
			Usage: "启动配置转换服务.",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "addr",
					Usage: "监听的地址和端口号",
					Value: ":10200",
				},
				&cli.StringFlag{
					Name:  "name",
					Usage: "配置名称",
					Value: "",
				},
				&cli.StringFlag{
					Name:  "converter",
					Usage: "转换模式",
					Value: "vmess",
				},
				&cli.StringFlag{
					Name:  "url",
					Usage: "配置文件地址",
					Value: "",
				},
				&cli.IntFlag{
					Name:  "interval",
					Usage: "自动更新频率,单位分钟",
					Value: 60,
				},
			},
			Action: func(c *cli.Context) error {
				if name := c.String("name"); name != "" {
					if urlStr := c.String("url"); urlStr != "" {
						err := server.AddVmess(name, c.String("converter"), urlStr, c.Int("interval"))
						if err != nil {
							log.Printf("添加配置失败 -> %s  %s\n", name, urlStr)
						}
					}
				}
				return server.Run(context.Background(), c.String("addr"))
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln("启动服务失败 ->", err)
	}
}
func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
