import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import Link from "next/link";
import {
  Reveal,
  StaggerGroup,
  StaggerItem,
} from "@/components/motion-primitives";

export default function FAQs() {
  const faqItems = [
    {
      id: "item-1",
      question: "部署 Pixoma 需要做好什么准备？",
      answer:
        "Pixoma 设计都支持依赖最小化（SQLite+本地文件存储），在 ALL IN ONE 场景下，你只需一台电脑即可完成整套服务的部署。但是对于网络环境受限的用户来说，还需要具备可正常访问 Telegram 的网络环境。",
    },
    {
      id: "item-2",
      question: "Pixoma 会帮我准备好 ComfyUI 吗?",
      answer:
        "不会，Pixoma 目前定位更多是帮助你把 ComfyUI 能力向外提供，所以需要您先准备好 ComfyUI 的运行环境，并且确保你的工作流都是可以正常运行。",
    },
    {
      id: "item-3",
      question: "如果我想要使用云端的 ComfyUI 计算节点，需要做哪些准备？",
      answer:
        "1. 把 Pixoma 进程部署到云服务器：需要具备公网 IP 或者域名，以便 Pixoma-Edge-Agent 可以访问 Pixoma 服务拉取工作流任务。<br/>2. 准备一个公网 OSS 云存储：用于存储工作流任务的输入文件以及输出结果。",
    },
  ];

  return (
    <section className="py-16 md:py-24">
      <div className="mx-auto max-w-7xl px-6">
        <Reveal className="grid gap-12 md:grid-cols-2 md:gap-6">
          <h2 className="text-foreground max-w-sm text-balance text-4xl font-medium tracking-tight">
            常见问题
          </h2>

          <div>
            <StaggerGroup viewportMargin="0px 0px -25% 0px">
              <Accordion className="w-full">
                {faqItems.map((item) => (
                  <StaggerItem
                    key={item.id}
                    className="border-b border-dashed last:border-b-0"
                  >
                    <AccordionItem value={item.id}>
                      <AccordionTrigger className="cursor-pointer text-base hover:no-underline">
                        {item.question}
                      </AccordionTrigger>
                      <AccordionContent>
                        <p className="text-muted-foreground text-base">
                          <span dangerouslySetInnerHTML={{ __html: item.answer }} />
                        </p>
                      </AccordionContent>
                    </AccordionItem>
                  </StaggerItem>
                ))}
              </Accordion>
            </StaggerGroup>

            <p className="text-muted-foreground mt-6">
              有问题需要反馈或者咨询? 欢迎提{" "}
              <Link
                href="https://github.com/Mr9esx/Pixoma/issues"
                target="_blank"
                className="text-primary font-medium hover:underline"
              >
                Issue
              </Link>
              。
            </p>
          </div>
        </Reveal>
      </div>
    </section>
  );
}
