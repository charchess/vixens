import { readWiki, countBy } from '$lib/wiki';

const GTD = process.env.GTDWIKI_PATH || '/data/gtdwiki';
const LLM = process.env.LLMWIKI_PATH || '/data/llmwiki';

export async function load() {
  const [gtdItems, llmItems] = await Promise.all([
    readWiki(GTD, 120).catch((error) => ({ error: String(error), items: [] })),
    readWiki(LLM, 60).catch((error) => ({ error: String(error), items: [] }))
  ]);

  const gtd = Array.isArray(gtdItems) ? gtdItems : gtdItems.items;
  const llm = Array.isArray(llmItems) ? llmItems : llmItems.items;

  return {
    generatedAt: new Date().toISOString(),
    paths: { gtd: GTD, llm: LLM },
    links: {
      llmwiki: 'http://192.168.199.119:4567/Home',
      gtdwiki: 'http://192.168.199.119:4568/Home'
    },
    gtd,
    llm,
    counts: {
      gtdTotal: gtd.length,
      llmTotal: llm.length,
      gtdByStatus: countBy(gtd, 'status'),
      gtdByOwner: countBy(gtd, 'owner')
    },
    errors: {
      gtd: Array.isArray(gtdItems) ? null : gtdItems.error,
      llm: Array.isArray(llmItems) ? null : llmItems.error
    }
  };
}
