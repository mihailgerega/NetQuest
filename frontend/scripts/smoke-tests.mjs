import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const root = resolve(process.cwd());

function read(path) {
  return readFileSync(resolve(root, path), "utf8");
}

function assertContains(file, text, message) {
  const source = read(file);
  if (!source.includes(text)) {
    throw new Error(`${message}: ${file} should contain ${JSON.stringify(text)}`);
  }
}

function assertNotContains(file, text, message) {
  const source = read(file);
  if (source.includes(text)) {
    throw new Error(`${message}: ${file} should not contain ${JSON.stringify(text)}`);
  }
}

assertContains("src/app/page.tsx", "NetQuest — визуальная лаборатория компьютерных сетей", "Landing hero is incomplete");
assertContains("src/app/page.tsx", "Что можно делать в NetQuest", "Landing capabilities section is missing");
assertContains("src/app/page.tsx", "Как проходит HTTPS-запрос", "Landing HTTPS flow section is missing");
assertContains("src/app/page.tsx", "Элементы сети", "Landing network elements section is missing");
assertContains("src/app/page.tsx", "Как считается время", "Landing timing section is missing");
assertContains("src/app/page.tsx", "Как начать", "Landing quick start section is missing");

assertContains("src/app/quests/page.tsx", "topicFilter", "Quest topic filter is missing");
assertContains("src/app/quests/page.tsx", "progressByDifficulty", "Quest progress by difficulty is missing");
assertContains("src/app/quests/page.tsx", "compareQuests", "Quest deterministic sorting is missing");
assertContains("src/app/quests/page.tsx", "Все темы", "Quest topic filter label is missing");

assertContains("src/app/simulator/page.tsx", "netquest.simulator.viewport", "Canvas viewport persistence is missing");
assertContains("src/app/simulator/page.tsx", "По размеру", "Canvas fit-to-view control is missing");
assertContains("src/app/simulator/page.tsx", "Сброс", "Canvas reset control is missing");
assertContains("src/app/simulator/page.tsx", "Инспектор пакета", "Packet inspector label is not localized");
assertContains("src/app/simulator/page.tsx", "Протокольный разбор", "Protocol inspector label is not localized");
assertNotContains("src/app/simulator/page.tsx", "Break Backend", "Old hardcoded break backend action leaked back");
assertNotContains("src/app/simulator/page.tsx", "link-lb-server-1", "Simulator still references demo link hardcode");

console.log("frontend smoke tests passed");
