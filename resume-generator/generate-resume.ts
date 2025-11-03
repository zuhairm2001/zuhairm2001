import fs from "fs";
import { marked } from "marked";
import puppeteer from "puppeteer";
import { S3Client, PutObjectCommand } from "@aws-sdk/client-s3";

const markdown = fs.readFileSync("resume.md", "utf-8");
const htmlContent = marked(markdown);
const css = fs.readFileSync("resume.css");

const fullHTML = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>zuhair's resume</title>
  <style>${css}</style>
</head>
<body class="page">
  ${htmlContent}
</body>
</html>
`;

(async () => {
  const browser = await puppeteer.launch();
  const page = await browser.newPage();
  await page.setContent(fullHTML, { waitUntil: "load" });
  await page.pdf({
    path: "output.pdf",
    format: "A4",
    printBackground: true,
  });
  await browser.close();
  console.log("PDF generated: output.pdf");
})();

const endpoint = Bun.env.S3_ENDPOINT!;
const accessKeyId = Bun.env.S3_ACCESS_KEY!;
const secretAccessKey = Bun.env.S3_SECRET_KEY!;

const S3 = new S3Client({
  region: "auto",
  endpoint: endpoint,
  credentials: {
    accessKeyId: accessKeyId,
    secretAccessKey: secretAccessKey,
  },
});

try {
  await S3.send(
    new PutObjectCommand({
      Bucket: "personal-bucket",
      Key: "resume.pdf",
      Body: fs.readFileSync("output.pdf"),
      ContentType: "application/pdf",
    }),
  );
} catch (error) {
  console.error("Error uploading file to S3:", error);
}
