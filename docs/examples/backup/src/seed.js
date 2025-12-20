/**
 * Database Seed Script
 *
 * Populates the database with sample data for testing backups.
 */

const { Pool } = require('pg');

const pool = new Pool({
  connectionString: process.env.DATABASE_URL || 'postgres://postgres:postgres@postgres:5432/backup-example_dev'
});

const sampleNotes = [
  {
    title: 'Welcome to Backup Example',
    content: 'This is a sample note to demonstrate database backups with dockstart.'
  },
  {
    title: 'Important Meeting Notes',
    content: 'Discuss project timeline, assign tasks to team members, review budget allocation.'
  },
  {
    title: 'Shopping List',
    content: '- Milk\n- Bread\n- Eggs\n- Coffee\n- Butter'
  },
  {
    title: 'Book Recommendations',
    content: '1. The Pragmatic Programmer\n2. Clean Code\n3. Design Patterns\n4. Refactoring'
  },
  {
    title: 'Project Ideas',
    content: 'Build a CLI tool that generates Docker configurations automatically. Call it dockstart!'
  }
];

async function seed() {
  const client = await pool.connect();
  try {
    // Create table if not exists
    await client.query(`
      CREATE TABLE IF NOT EXISTS notes (
        id SERIAL PRIMARY KEY,
        title VARCHAR(255) NOT NULL,
        content TEXT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
      )
    `);

    // Clear existing data
    await client.query('TRUNCATE notes RESTART IDENTITY');

    // Insert sample notes
    for (const note of sampleNotes) {
      await client.query(
        'INSERT INTO notes (title, content) VALUES ($1, $2)',
        [note.title, note.content]
      );
    }

    console.log(`Seeded ${sampleNotes.length} notes into the database`);

    // Verify data
    const result = await client.query('SELECT COUNT(*) FROM notes');
    console.log(`Total notes in database: ${result.rows[0].count}`);

  } finally {
    client.release();
    await pool.end();
  }
}

seed().catch(err => {
  console.error('Seed failed:', err);
  process.exit(1);
});
