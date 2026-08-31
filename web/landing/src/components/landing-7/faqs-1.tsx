import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/landing-7/ui/accordion";
import Link from "next/link";

export default function FAQs() {
  const faqItems = [
    {
      id: "item-1",
      question: "Pixoma 是什么？",
      answer:
        "Pixoma 让你随时随地使用自己的 ComfyUI：通过 Telegram Bot 发起创作，由 Case 目录与 Task 运行时负责生成和交付。",
    },
    {
      id: "item-2",
      question: "必须使用云端 ComfyUI 吗？",
      answer:
        "不需要。你可以连接本地、内网或云端节点，Pixoma 会按任务状态与节点健康情况调度执行。",
    },
    {
      id: "item-3",
      question: "支持哪些工作流？",
      answer:
        "你可以把 ComfyUI Workflow 保存为 Case，配置输入项、路由与绑定，之后在对话中直接复用。",
    },
    {
      id: "item-4",
      question: "生成结果保存在哪里？",
      answer:
        "任务产物会进入对象存储，并回传到对应会话，方便你查看、复用和归档。",
    },
    {
      id: "item-5",
      question: "项目是开源的吗？",
      answer:
        "是的，Pixoma 正在开源迭代，你可以查看源码、提 Issue 或参与贡献。",
    },
  ];

  return (
    <section className="py-16 md:py-24">
      <div className="mx-auto max-w-7xl px-6">
        <div className="grid gap-12 md:grid-cols-2 md:gap-6">
          <h2 className="text-foreground max-w-sm text-balance text-4xl font-medium tracking-tight">
            常见问题
          </h2>

          <div>
            <Accordion className="w-full">
              {faqItems.map((item) => (
                <AccordionItem
                  key={item.id}
                  value={item.id}
                  className="border-dashed"
                >
                  <AccordionTrigger className="cursor-pointer text-base hover:no-underline">
                    {item.question}
                  </AccordionTrigger>
                  <AccordionContent>
                    <p className="text-muted-foreground text-base">
                      {item.answer}
                    </p>
                  </AccordionContent>
                </AccordionItem>
              ))}
            </Accordion>

            <p className="text-muted-foreground mt-6">
              还有其他问题？欢迎在{" "}
              <Link
                href="https://github.com/Mr9esx/Pixoma"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary font-medium hover:underline"
              >
                GitHub 仓库
              </Link>
              提问。
            </p>
          </div>
        </div>
      </div>
    </section>
  );
}
