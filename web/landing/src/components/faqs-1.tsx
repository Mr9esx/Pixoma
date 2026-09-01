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
      question: "How long does shipping take?",
      answer:
        "Standard shipping takes 3-5 business days, depending on your location. Express shipping options are available at checkout for 1-2 business day delivery.",
    },
    {
      id: "item-2",
      question: "What payment methods do you accept?",
      answer:
        "We accept all major credit cards (Visa, Mastercard, American Express), PayPal, Apple Pay, and Google Pay. For enterprise customers, we also offer invoicing options.",
    },
    {
      id: "item-3",
      question: "Can I change or cancel my order?",
      answer:
        "You can modify or cancel your order within 1 hour of placing it. After this window, please contact our customer support team who will assist you with any changes.",
    },
    {
      id: "item-4",
      question: "Do you ship internationally?",
      answer:
        "Yes, we ship to over 50 countries worldwide. International shipping typically takes 7-14 business days. Additional customs fees may apply depending on your country's import regulations.",
    },
    {
      id: "item-5",
      question: "What is your return policy?",
      answer:
        "We offer a 30-day return policy for most items. Products must be in original condition with tags attached. Some specialty items may have different return terms, which will be noted on the product page.",
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
                          {item.answer}
                        </p>
                      </AccordionContent>
                    </AccordionItem>
                  </StaggerItem>
                ))}
              </Accordion>
            </StaggerGroup>

            <p className="text-muted-foreground mt-6">
              Can't find what you're looking for? Contact our{" "}
              <Link
                href="#"
                className="text-primary font-medium hover:underline"
              >
                customer support team
              </Link>
            </p>
          </div>
        </Reveal>
      </div>
    </section>
  );
}
