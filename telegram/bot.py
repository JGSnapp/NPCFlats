import os

from telegram import InlineKeyboardButton, InlineKeyboardMarkup, Update, WebAppInfo
from telegram.ext import Application, CommandHandler, ContextTypes

async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    keyboard = [
        [InlineKeyboardButton(text="В игру 🚪", web_app=WebAppInfo(url="https://d-art.space"))]
    ]
    reply_markup = InlineKeyboardMarkup(keyboard)
    await update.message.reply_text("Buy, develop and sell apartments in a new exciting real estate owner simulator! \n \n Покупай, обустраивай и продавай квартиры в новом захватывающем симуляторе владельца недвижимости!", reply_markup=reply_markup)

def main():
    application = Application.builder().token(os.environ["BOT_TOKEN"]).build()
    start_handler = CommandHandler('start', start)
    application.add_handler(start_handler)
    application.run_polling()

if __name__ == '__main__':
    main()
