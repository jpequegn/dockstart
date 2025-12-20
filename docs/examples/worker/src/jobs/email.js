/**
 * Email job processor
 *
 * In a real application, this would send emails via
 * SendGrid, Mailgun, SES, etc.
 */

async function processEmail(job) {
  const { to, subject, body, createdAt } = job.data;

  console.log(`Sending email to: ${to}`);
  console.log(`Subject: ${subject}`);

  // Update progress
  await job.updateProgress(10);

  // Simulate email sending delay
  await new Promise(resolve => setTimeout(resolve, 1000));

  await job.updateProgress(50);

  // In production, you would call your email service here:
  // await sendgrid.send({ to, subject, body });

  await new Promise(resolve => setTimeout(resolve, 500));

  await job.updateProgress(100);

  console.log(`Email sent successfully to ${to}`);

  return {
    sent: true,
    to,
    subject,
    sentAt: new Date().toISOString()
  };
}

module.exports = { processEmail };
