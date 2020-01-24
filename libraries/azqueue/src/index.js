const taskcluster = require('taskcluster-client');

class AZQueue {
  constructor({ db }) {
    this.db = db;
  }

  async createQueue(name, metadata) {
    // NOOP
  }

  async getMetadata(name) {
    const result = await this.db.fns.azure_queue_count(name);

    return {
      messageCount: result[0].azure_queue_count,
    };
  }

  async setMetadata(name, update) {
    // NOOP
  }

  async putMessage(name, text, {visibilityTimeout, messageTTL}) {
    await this.db.fns.azure_queue_put(
      name,
      text,
      taskcluster.fromNow(`${visibilityTimeout} seconds`),
      taskcluster.fromNow(`${messageTTL} seconds`),
    );
  }

  async getMessages(name, {visibilityTimeout, numberOfMessages}) {
    // TODO: how to listen, and for how long..
    // TODO: this should return a cancel-able promise?
    const res = await this.db.fns.azure_queue_get(
      name,
      taskcluster.fromNow(`${visibilityTimeout} seconds`),
      numberOfMessages);
    return res.map(({message_id, message_text, pop_receipt}) => ({
      messageId: message_id,
      messageText: message_text,
      popReceipt: pop_receipt,
    }));
  }

  async deleteMessage(name, messageId, popReceipt) {

  }

  async updateMessage(name, messageText, messageId, popRecipt, {visibilityTimeout}) {

  }

  async listQueues() {
    // stubbed out
    return {queues: []};
  }

  async deleteQueue(name) {
    // NOOP
  }
}

module.exports = AZQueue;